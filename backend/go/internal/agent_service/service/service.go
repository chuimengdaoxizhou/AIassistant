package service

import (
	v1 "Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/agent"
	"Jarvis_2.0/backend/go/internal/database/kafka"
	"Jarvis_2.0/backend/go/internal/llm"
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sync"
	"time"
)

const maxIterations = 10 // 定义 ReAct 循环的最大次数

// AgentService 封装了 MainAgent 的核心逻辑。
type AgentService struct {
	llmClient    llm.LLM
	registry     *agent.Registry // 使用单例 Registry 指针
	logPublisher *kafka.LogPublisher
	taskStore    TaskStore
}

// NewAgentService 创建一个新的 AgentService 实例。
func NewAgentService(llmClient llm.LLM, registry *agent.Registry, logPublisher *kafka.LogPublisher, taskStore TaskStore) *AgentService {
	return &AgentService{
		llmClient:    llmClient,
		registry:     registry,
		logPublisher: logPublisher,
		taskStore:    taskStore,
	}
}

// Execute 实现 agent.Agent gRPC 接口，它现在是 RunReActLoop 的一个包装器。
func (s *AgentService) Execute(ctx context.Context, task *v1.AgentTask) (*v1.AgentTask, error) {
	return s.RunReActLoop(ctx, task)
}

// RunReActLoop 包含完整的 ReAct 核心循环逻辑。
func (s *AgentService) RunReActLoop(ctx context.Context, task *v1.AgentTask) (*v1.AgentTask, error) {
	appLogger := logger.New("main_agent_service", task.TaskId, "")
	appLogger.Info("Main agent starting ReAct loop")

	history := models.ConvertProtoToModelsContent(task.Content)

	for i := 0; i < maxIterations; i++ {
		appLogger.WithPayload(map[string]interface{}{"iteration": i + 1}).Info("ReAct loop iteration")
		s.logProgress(ctx, task.TaskId, task.CorrelationId, models.StatusThinking, fmt.Sprintf("循环 %d：Agent 正在思考...", i+1), history)

		llmReq := &models.GenerateContentRequest{
			Content: history,
		}

		llmResp, err := s.llmClient.GenerateContent(ctx, llmReq)
		if err != nil {
			appLogger.WithError(models.ErrorInfo{Message: err.Error()}).Error("LLM GenerateContent failed")
			s.logProgress(ctx, task.TaskId, task.CorrelationId, models.StatusError, "调用大语言模型失败", err.Error())
			return nil, err
		}

		if len(llmResp.Content) > 0 && llmResp.Content[0].HasFunctionCall() {
			appLogger.Info("LLM decided to use a tool.")
			modelRequestContent := llmResp.Content[0]
			history = append(history, modelRequestContent)

			observationContents, err := s.executeFunctionCalls(ctx, task, modelRequestContent.Parts)
			if err != nil {
				var mcpErr *MCPError
				if errors.As(err, &mcpErr) {
					return nil, mcpErr
				}
				appLogger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Function call execution failed")
				s.logProgress(ctx, task.TaskId, task.CorrelationId, models.StatusError, "工具调用执行失败", err.Error())
				return nil, err
			}

			history = append(history, observationContents...)
			continue
		} else {
			appLogger.Info("LLM provided final answer.")
			if len(llmResp.Content) == 0 {
				return nil, fmt.Errorf("LLM returned empty content")
			}

			finalContent := llmResp.Content[0]
			s.logProgress(ctx, task.TaskId, task.CorrelationId, models.StatusFinished, "任务已完成", finalContent)

			finalTask, err := models.ConvertModelsToProtoTask(task, finalContent, task.TargetAgentId, "Final Result")
			if err != nil {
				appLogger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to convert final content to proto task")
				return nil, err
			}
			return finalTask, nil
		}
	}

	appLogger.Error("Reached max iterations, stopping.")
	s.logProgress(ctx, task.TaskId, task.CorrelationId, models.StatusError, "任务达到最大循环次数", nil)
	return nil, fmt.Errorf("reached max iterations")
}

func (s *AgentService) logProgress(ctx context.Context, taskID, correlationID string, status models.TaskLogStatus, message string, content interface{}) {
	entry := &models.TaskLogEntry{
		TaskID:        taskID,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		Status:        status,
		Message:       message,
		Content:       content,
	}
	if err := s.logPublisher.LogTaskProgress(ctx, entry); err != nil {
		errorLogger := logger.New("log_publisher_error", taskID, correlationID)
		errorLogger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to send task log to kafka")
	}
}

func (s *AgentService) executeFunctionCalls(ctx context.Context, originalTask *v1.AgentTask, parts []*models.Part) ([]models.Content, error) {
	appLogger := logger.New("function_executor", originalTask.TaskId, originalTask.CorrelationId)
	var wg sync.WaitGroup
	observationChan := make(chan models.Content, len(parts))
	errChan := make(chan error, 1)

	for _, part := range parts {
		if part.FunctionCall == nil {
			continue
		}

		fc := part.FunctionCall

		// 使用 DiscoverAgents 动态发现子 Agent
		addresses, err := s.registry.DiscoverAgents(fc.Name)
		// 如果发现过程出错，或者没有找到任何地址，则判定为 MCP 工具
		if err != nil || len(addresses) == 0 {
			if err != nil {
				appLogger.WithError(models.ErrorInfo{Message: err.Error()}).Warn(fmt.Sprintf("Failed to discover agent '%s', assuming it is an MCP tool.", fc.Name))
			} else {
				appLogger.WithPayload(map[string]interface{}{"tool_name": fc.Name}).Warn("Tool not found via discovery. Assuming it is an MCP tool.")
			}

			s.logProgress(ctx, originalTask.TaskId, originalTask.CorrelationId, models.StatusCallingMCPTool, fmt.Sprintf("决定调用 MCP 工具: %s", fc.Name), fc)

			protoTask, err := models.ConvertModelsToProtoTask(originalTask, models.Content{Role: models.SpeakerAssistant, Parts: []*models.Part{{FunctionCall: fc}}}, "mcp_tool_handler", fc.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to convert MCP call to proto task: %w", err)
			}
			return nil, &MCPError{Task: protoTask}
		}

		// 如果找到了地址，则判定为子Agent，并异步执行
		wg.Add(1)
		go func(call *models.FunctionCall, addr string) {
			defer wg.Done()
			appLogger.Info(fmt.Sprintf("Handling sub-agent call: %s at %s", call.Name, addr))
			s.logProgress(ctx, originalTask.TaskId, originalTask.CorrelationId, models.StatusCallingSubAgent, fmt.Sprintf("决定调用子 Agent: %s", call.Name), call)

			var observation map[string]any

			// 动态建立gRPC连接
			conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				appLogger.WithError(models.ErrorInfo{Message: err.Error()}).Error(fmt.Sprintf("Failed to dial sub-agent %s", call.Name))
				observation = map[string]any{"error": fmt.Sprintf("failed to connect to sub-agent: %s", call.Name)}
			} else {
				defer conn.Close()
				client := v1.NewAgentServiceClient(conn)

				subTaskContent := models.Content{
					Role:  models.SpeakerUser,
					Parts: []*models.Part{{Text: call.ArgsToString()}},
				}

				subTask, err := models.ConvertModelsToProtoTask(originalTask, subTaskContent, call.Name, fmt.Sprintf("Sub-task for %s", call.Name))
				if err != nil {
					appLogger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to create sub-task")
					observation = map[string]any{"error": "failed to create sub-task"}
				} else {
					resultTask, err := client.ExecuteTask(ctx, subTask)
					if err != nil {
						appLogger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Sub-agent gRPC execution failed")
						observation = map[string]any{"error": err.Error()}
					} else {
						observation = map[string]any{"output": models.ConvertProtoToModelsContent(resultTask.Content)}
					}
				}
			}

			obsContent := models.Content{
				Role: models.SpeakerTool,
				Parts: []*models.Part{{
					FunctionResponse: &models.FunctionResponse{Name: call.Name, Response: observation},
				}},
			}
			s.logProgress(ctx, originalTask.TaskId, originalTask.CorrelationId, models.StatusObserving, fmt.Sprintf("从工具 %s 获得观察结果", call.Name), obsContent)
			observationChan <- obsContent
		}(fc, addresses[0]) // 使用发现的第一个地址
	}

	wg.Wait()
	close(observationChan)
	close(errChan)

	if err := <-errChan; err != nil {
		return nil, err
	}

	observationContents := make([]models.Content, 0)
	for obs := range observationChan {
		observationContents = append(observationContents, obs)
	}

	return observationContents, nil
}

func (s *AgentService) Metadata() agent.AgentMetadata {
	return agent.AgentMetadata{
		Name:       "main_agent_service",
		Capability: "Orchestrates tasks and coordinates sub-agents using a ReAct loop.",
	}
}

// MCPError 是一个特殊的错误类型，用于信号传递，表示需要调用MCP工具。
type MCPError struct {
	Task *v1.AgentTask
}

func (e *MCPError) Error() string {
	return fmt.Sprintf("MCP tool call required for task: %s", e.Task.TaskId)
}

// TaskStore 定义了与任务持久化存储交互的接口。
type TaskStore interface {
	GetTaskByID(ctx context.Context, taskID string) (*v1.AgentTask, error)
	GetTasksByParentID(ctx context.Context, parentID string) ([]*v1.AgentTask, error)
}

// --- 任务树相关功能保持不变 ---
type TaskNode struct {
	Task     *v1.AgentTask `json:"task"`
	Children []*TaskNode   `json:"children"`
}

func (s *AgentService) GetTaskTree(ctx context.Context, taskID string) (*TaskNode, error) {
	var rootTask *v1.AgentTask
	currentTaskID := taskID
	for {
		task, err := s.taskStore.GetTaskByID(ctx, currentTaskID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task '%s': %w", currentTaskID, err)
		}
		if task.ParentTaskId == "" {
			rootTask = task
			break
		}
		currentTaskID = task.ParentTaskId
	}
	return s.buildSubTree(ctx, rootTask)
}

func (s *AgentService) buildSubTree(ctx context.Context, task *v1.AgentTask) (*TaskNode, error) {
	node := &TaskNode{
		Task:     task,
		Children: []*TaskNode{},
	}
	childrenTasks, err := s.taskStore.GetTasksByParentID(ctx, task.TaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to get children for task '%s': %w", task.TaskId, err)
	}
	for _, childTask := range childrenTasks {
		childNode, err := s.buildSubTree(ctx, childTask)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, childNode)
	}
	return node, nil
}