package service

import (
	v1 "Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/agent"
	"Jarvis_2.0/backend/go/internal/database/kafka"
	"Jarvis_2.0/backend/go/internal/llm"
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const maxIterations = 10 // 定义 ReAct 循环的最大次数

// AgentService 封装了 MainAgent 的核心逻辑。
type AgentService struct {
	llmClient    llm.LLM
	registry     *agent.DistributedRegistry
	logPublisher *kafka.LogPublisher
	taskStore    TaskStore // 添加 TaskStore 依赖
}

// NewAgentService 创建一个新的 AgentService 实例。
func NewAgentService(llmClient llm.LLM, registry *agent.DistributedRegistry, logPublisher *kafka.LogPublisher, taskStore TaskStore) *AgentService {
	return &AgentService{
		llmClient:    llmClient,
		registry:     registry,
		logPublisher: logPublisher,
		taskStore:    taskStore,
	}
}

// logProgress 是一个辅助函数，用于创建并发送任务进度日志。
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
		// 日志发送失败不应阻塞主流程，但需要记录下来
		appLogger := logger.New("log_publisher_error", taskID, correlationID)
		appLogger.Error(fmt.Sprintf("Failed to send task log to kafka: %v", err))
	}
}

// Metadata 实现 agent.Agent 接口，返回 MainAgent 的元数据。
func (s *AgentService) Metadata() agent.AgentMetadata {
	return agent.AgentMetadata{
		Name:              "main_agent",
		Capability:        "作为一个协调器，理解用户任务并调用合适的子 Agent 来完成任务。",
		InputDescription:  "包含用户原始请求的 AgentTask。",
		OutputDescription: "包含任务最终结果或需要上层处理的 FunctionCall 的 AgentTask。",
	}
}

// Execute 实现 agent.Agent 接口，包含使用 Function Calling 的 ReAct 核心循环。
func (s *AgentService) Execute(ctx context.Context, task *v1.AgentTask) (*v1.AgentTask, error) {
	appLogger := logger.New("main_agent_service", task.TaskId, task.CorrelationId)
	appLogger.Info("Main agent service received task")

	history := models.ConvertProtoToModelsContent(task.Content)

	for i := 0; i < maxIterations; i++ {
		appLogger.Info(fmt.Sprintf("ReAct loop iteration: %d", i+1))
		s.logProgress(ctx, task.TaskId, task.CorrelationId, models.StatusThinking, fmt.Sprintf("循环 %d：Agent 正在思考...", i+1), history)

		llmReq := &models.GenerateContentRequest{
			Content: history,
		}
		llmReq.Content = append([]models.Content{s.buildToolPrompt()}, llmReq.Content...)

		llmResp, err := s.llmClient.GenerateContent(ctx, llmReq)
		if err != nil {
			appLogger.Error(fmt.Sprintf("LLM GenerateContent failed: %v", err))
			s.logProgress(ctx, task.TaskId, task.CorrelationId, models.StatusError, "调用大语言模型失败", err.Error())
			return nil, err
		}

		hasFunctionCall := false
		for _, part := range llmResp.Content[0].Parts {
			if part.FunctionCall != nil {
				hasFunctionCall = true
				break
			}
		}

		if hasFunctionCall {
			modelRequestContent, observationContents, err := s.executeFunctionCalls(ctx, task, llmResp.Content[0].Parts)
			if err != nil {
				// 如果是 MCP 工具调用，则直接返回错误，因为需要上层处理
				return nil, err
			}
			history = append(history, modelRequestContent)
			history = append(history, observationContents...)
			continue
		} else {
			finalAnswer := llmResp.Content[0].Parts[0].Text
			appLogger.Info(fmt.Sprintf("LLM provided final answer: %s", finalAnswer))
			s.logProgress(ctx, task.TaskId, task.CorrelationId, models.StatusFinished, "任务已完成", finalAnswer)
			return s.createFinalResponse(task, finalAnswer)
		}
	}

	appLogger.Error("Reached max iterations, stopping.")
	s.logProgress(ctx, task.TaskId, task.CorrelationId, models.StatusError, "任务达到最大循环次数", nil)
	return nil, fmt.Errorf("reached max iterations")
}

func (s *AgentService) executeFunctionCalls(ctx context.Context, originalTask *v1.AgentTask, parts []*models.Part) (models.Content, []models.Content, error) {
	appLogger := logger.New("function_executor", originalTask.TaskId, originalTask.CorrelationId)
	var wg sync.WaitGroup
	observationChan := make(chan models.Content, len(parts))
	modelRequestParts := make([]*models.Part, 0)

	for _, part := range parts {
		if part.FunctionCall == nil {
			continue
		}
		modelRequestParts = append(modelRequestParts, &models.Part{FunctionCall: part.FunctionCall})
		fc := part.FunctionCall

		// 在 goroutine 外检查 agent 是否存在
		client, found := s.registry.GetAgentClient(fc.Name)
		if !found {
			// 是 MCP 工具调用，我们不处理，直接返回错误，让上层处理
			appLogger.Info(fmt.Sprintf("Passing MCP tool call '%s' to upstream", fc.Name))
			s.logProgress(ctx, originalTask.TaskId, originalTask.CorrelationId, models.StatusCallingMCPTool, fmt.Sprintf("决定调用 MCP 工具: %s", fc.Name), fc)
			protoTask, err := models.ConvertModelsToProtoTask(originalTask, models.Content{Role: models.SpeakerAssistant, Parts: []*models.Part{{FunctionCall: fc}}}, "mcp_tool_handler", fc.Name)
			if err != nil {
				return models.Content{}, nil, fmt.Errorf("failed to convert MCP call to proto task: %w", err)
			}
			return models.Content{}, nil, &MCPError{Task: protoTask} // 使用自定义错误类型传递任务
		}

		wg.Add(1)
		go func(call *models.FunctionCall, client v1.AgentServiceClient) {
			defer wg.Done()
			appLogger.Info(fmt.Sprintf("Handling sub-agent call: %s", call.Name))
			s.logProgress(ctx, originalTask.TaskId, originalTask.CorrelationId, models.StatusCallingSubAgent, fmt.Sprintf("决定调用子 Agent: %s", call.Name), call)

			var observation map[string]any
			appLogger.Info(fmt.Sprintf("Executing remote sub-agent: %s", call.Name))
			inputBytes, _ := json.Marshal(call.Args)

			subTask, err := models.ConvertModelsToProtoTask(originalTask, models.Content{Parts: []*models.Part{{Text: string(inputBytes)}}}, call.Name, fmt.Sprintf("Sub-task for %s", call.Name))
			if err != nil {
				appLogger.Error(fmt.Sprintf("Failed to create sub-task: %v", err))
				observation = map[string]any{"error": "failed to create sub-task"}
			} else {
				resultTask, err := client.ExecuteTask(ctx, subTask)
				if err != nil {
					appLogger.Error(fmt.Sprintf("Sub-agent gRPC execution failed: %v", err))
					observation = map[string]any{"error": err.Error()}
				} else {
					observation = map[string]any{"output": resultTask.Content[0].Parts[0].GetText()}
				}
			}

			// 将观察结果发送到通道
			obsContent := models.Content{
				Role: models.SpeakerTool,
				Parts: []*models.Part{{
					FunctionResponse: &models.FunctionResponse{Name: call.Name, Response: observation},
				}},
			}
			s.logProgress(ctx, originalTask.TaskId, originalTask.CorrelationId, models.StatusObserving, fmt.Sprintf("从工具 %s 获得观察结果", call.Name), obsContent)
			observationChan <- obsContent
		}(fc, client)
	}

	wg.Wait()
	close(observationChan)

	observationContents := make([]models.Content, 0)
	for obs := range observationChan {
		observationContents = append(observationContents, obs)
	}

	return models.Content{Role: models.SpeakerModel, Parts: modelRequestParts}, observationContents, nil
}

// MCPError 是一个自定义错误类型，用于向上传递需要处理的 MCP 工具调用。
type MCPError struct {
	Task *v1.AgentTask
}

func (e *MCPError) Error() string {
	return fmt.Sprintf("MCP tool call requires upstream handling: %s", e.Task.Content[0].Parts[0].FunctionCall.Name)
}

func (s *AgentService) buildToolPrompt() models.Content {
	var sb strings.Builder
	sb.WriteString("You have access to the following tools. Use them if necessary. To use a tool, respond with a FunctionCall.\n")
	sb.WriteString("Available Tools:\n")

	tools := s.registry.GetAgentMetadata()
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("- Function Name: %s\n", tool.Name))
		sb.WriteString(fmt.Sprintf("  Description: %s\n", tool.Capability))
		sb.WriteString(fmt.Sprintf("  Input: %s\n", tool.InputDescription))
		sb.WriteString(fmt.Sprintf("  Output: %s\n", tool.OutputDescription))
	}

	return models.Content{
		Role:  models.SpeakerUser,
		Parts: []*models.Part{{Text: sb.String()}},
	}
}

func (s *AgentService) createFinalResponse(originalTask *v1.AgentTask, finalAnswer string) (*v1.AgentTask, error) {
	return models.ConvertModelsToProtoTask(originalTask, models.Content{Parts: []*models.Part{{Text: finalAnswer}}}, originalTask.SourceAgentId, "Final Result")
}

// --- 新增：任务树构建功能 ---

// TaskNode 表示任务树中的一个节点。
type TaskNode struct {
	Task     *v1.AgentTask `json:"task"`
	Children []*TaskNode   `json:"children"`
}

// TaskStore 是一个接口，定义了从数据存储中检索任务所需的方法。
// 注意：这只是一个接口定义。您需要在您的数据库层（例如mongo, mysql）中提供一个具体的实现。
type TaskStore interface {
	GetTaskByID(ctx context.Context, taskID string) (*v1.AgentTask, error)
	GetTasksByParentID(ctx context.Context, parentID string) ([]*v1.AgentTask, error)
}

// GetTaskTree 通过任何一个任务ID，构建并返回其所属的完整任务树。
func (s *AgentService) GetTaskTree(ctx context.Context, taskID string) (*TaskNode, error) {
	// 1. 从给定的 taskID 开始，向上遍历找到根任务
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

	// 2. 从根任务开始，递归地构建整个树
	return s.buildSubTree(ctx, rootTask)
}

// buildSubTree 是一个递归辅助函数，用于构建以给定任务为根的子树。
func (s *AgentService) buildSubTree(ctx context.Context, task *v1.AgentTask) (*TaskNode, error) {
	node := &TaskNode{
		Task:     task,
		Children: []*TaskNode{},
	}

	childrenTasks, err := s.taskStore.GetTasksByParentID(ctx, task.TaskId)
	if err != nil {
		// 如果只是找不到子任务，可以不视为错误，而是当作叶子节点
		// 但如果是其他数据库错误，则应返回错误
		// 此处为简化，我们假设找不到子任务会返回空切片和 nil 错误
		return nil, fmt.Errorf("failed to get children for task '%s': %w", task.TaskId, err)
	}

	for _, childTask := range childrenTasks {
		childNode, err := s.buildSubTree(ctx, childTask)
		if err != nil {
			return nil, err // 向上冒泡错误
		}
		node.Children = append(node.Children, childNode)
	}

	return node, nil
}
