package service

import (
	v1 "Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/agent"
	"Jarvis_2.0/backend/go/internal/llm"
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"os/exec"
	"strings"
)

const maxIterations = 10 // ReAct 循环的最大次数

// DataAnalysisService 实现了数据分析 Agent 的核心业务逻辑。
type DataAnalysisService struct {
	llm    llm.LLM
	logger *logger.Logger
}

// NewDataAnalysisService 创建一个新的 DataAnalysisService 实例。
func NewDataAnalysisService(llm llm.LLM, logger *logger.Logger) *DataAnalysisService {
	return &DataAnalysisService{
		llm:    llm,
		logger: logger,
	}
}

// Metadata 返回数据分析 Agent 的元数据。
func (s *DataAnalysisService) Metadata() agent.AgentMetadata {
	return agent.AgentMetadata{
		Name:              "data_analyzer",
		Capability:        "对指定的文件（如CSV）进行深入的数据分析，并回答相关问题。",
		InputDescription:  "一个JSON对象，必须包含 'file_path' (字符串) 和 'question' (字符串) 两个键。",
		OutputDescription: "一个包含分析结果最终答案的字符串。",
	}
}

// Execute 执行数据分析任务，内部包含一个 ReAct 循环。
func (s *DataAnalysisService) Execute(ctx context.Context, task *v1.AgentTask) (*v1.AgentTask, error) {
	s.logger.Info("Data analysis agent received a task")

	// 1. 解析输入参数
	var args struct {
		FilePath string `json:"file_path"`
		Question string `json:"question"`
	}
	if err := json.Unmarshal([]byte(task.Content[0].Parts[0].GetText()), &args); err != nil {
		return nil, fmt.Errorf("invalid input arguments: %w", err)
	}
	s.logger.Info(fmt.Sprintf("Task details: FilePath=%s, Question=%s", args.FilePath, args.Question))

	// 2. 初始化 ReAct 循环
	history := []models.Content{
		{
			Role: models.SpeakerUser,
			Parts: []*models.Part{{
				Text: fmt.Sprintf("我的任务是基于文件 '%s' 回答问题：'%s'。", args.FilePath, args.Question),
			}},
		},
	}

	for i := 0; i < maxIterations; i++ {
		s.logger.Info(fmt.Sprintf("ReAct loop iteration: %d", i+1))

		// 3. 思考 (Reason)
		llmReq := &models.GenerateContentRequest{
			Content: history,
		}
		// 将工具定义添加到 Prompt
		llmReq.Content = append([]models.Content{s.buildToolPrompt()}, llmReq.Content...)

		llmResp, err := s.llm.GenerateContent(ctx, llmReq)
		if err != nil {
			s.logger.Error(fmt.Sprintf("LLM GenerateContent failed: %v", err))
			return nil, err
		}

		// 检查 LLM 的回复是否是直接答案
		part := llmResp.Content[0].Parts[0]
		if part.FunctionCall == nil {
			s.logger.Info("LLM provided the final answer.")
			return s.createFinalResponse(task, part.Text)
		}

		// 4. 行动 (Act)
		s.logger.Info(fmt.Sprintf("LLM requested to call tool: %s", part.FunctionCall.Name))
		observation, err := s.executeTool(ctx, part.FunctionCall)
		if err != nil {
			s.logger.Error(fmt.Sprintf("Tool execution failed: %v", err))
			// 将错误作为观察结果反馈给 LLM
			observation = fmt.Sprintf("Error executing tool %s: %v", part.FunctionCall.Name, err)
		}

		// 5. 观察 (Observe)
		// 将本次的函数调用和观察结果添加到历史记录中
		history = append(history, models.Content{Role: models.SpeakerModel, Parts: []*models.Part{part}})
		history = append(history, models.Content{
			Role: models.SpeakerTool,
			Parts: []*models.Part{{
				FunctionResponse: &models.FunctionResponse{
					Name:     part.FunctionCall.Name,
					Response: map[string]any{"output": observation},
				},
			}},
		})
	}

	s.logger.Error("Reached max iterations, stopping.")
	return nil, fmt.Errorf("reached max iterations")
}

// buildToolPrompt 创建并返回一个包含可用工具描述的 Prompt。
func (s *DataAnalysisService) buildToolPrompt() models.Content {
	var sb strings.Builder
	sb.WriteString("你是一个数据分析专家。你可以使用以下工具来帮助你分析文件并回答问题。\n")
	sb.WriteString("可用的工具：\n")
	sb.WriteString("- 函数名: get_summary\n")
	sb.WriteString("  描述: 读取一个数据文件（如CSV），并返回其摘要信息，包括列名、数据类型和行数。\n")
	sb.WriteString("  输入: 一个JSON对象，包含 'file_path' (字符串) 键。\n")
	sb.WriteString("  输出: 一个包含摘要信息的JSON字符串。\n")
	// 在这里可以添加更多工具的描述，例如 run_pandas_query

	return models.Content{
		Role:  models.SpeakerUser,
		Parts: []*models.Part{{Text: sb.String()}},
	}
}

// executeTool 调用相应的 Python MCP 工具。
func (s *DataAnalysisService) executeTool(ctx context.Context, fc *models.FunctionCall) (string, error) {
	// 目前只实现 get_summary
	if fc.Name != "get_summary" {
		return "", fmt.Errorf("tool '%s' is not supported", fc.Name)
	}

	filePath, ok := fc.Args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("invalid or missing 'file_path' argument for get_summary")
	}

	// 构建命令
	// 注意：这里的路径是相对于项目根目录的
	cmd := exec.CommandContext(ctx, "python3", "backend/python/tools/data_analyzer/get_summary.py", "--file-path", filePath)

	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run get_summary.py: %w, stderr: %s", err, errOut.String())
	}

	return out.String(), nil
}

// createFinalResponse 创建并返回一个包含最终答案的 AgentTask。
func (s *DataAnalysisService) createFinalResponse(originalTask *v1.AgentTask, finalAnswer string) (*v1.AgentTask, error) {
	return &v1.AgentTask{
		TaskId:        uuid.New().String(),
		CorrelationId: originalTask.CorrelationId,
		ParentTaskId:  originalTask.TaskId,
		SourceAgentId: s.Metadata().Name,
		TargetAgentId: originalTask.SourceAgentId,
		TaskName:      "Data Analysis Result",
		Content: []*v1.Content{
			{
				Parts: []*v1.Part{
					{Text: finalAnswer},
				},
			},
		},
	}, nil
}
