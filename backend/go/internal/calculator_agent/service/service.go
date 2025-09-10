package service

import (
	v1 "Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/agent"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"strconv"
	"strings"
)

// CalculatorService 实现了计算器的核心业务逻辑。
type CalculatorService struct{}

// NewCalculatorService 创建一个新的 CalculatorService 实例。
func NewCalculatorService() *CalculatorService {
	return &CalculatorService{}
}

// Metadata 返回计算器 Agent 的元数据。
func (s *CalculatorService) Metadata() agent.AgentMetadata {
	return agent.AgentMetadata{
		Name:                "calculator",
		Capability:          "执行基本的数学运算，例如加、减、乘、除。",
		InputDescription:    "一个需要计算的数学表达式字符串，例如 '1 + 2' 或 '10 * 5'。",
		OutputDescription:   "一个包含计算结果的字符串。",
	}
}

// Execute 执行计算任务。
func (s *CalculatorService) Execute(ctx context.Context, task *v1.AgentTask) (*v1.AgentTask, error) {
	appLogger := logger.New("calculator_agent", task.CorrelationId, "")
	appLogger.Info("Calculator agent received task")

	var expression string
	if len(task.Content) > 0 && len(task.Content[0].Parts) > 0 {
		var args struct {
			Input string `json:"input"`
		}
		if err := json.Unmarshal([]byte(task.Content[0].Parts[0].GetText()), &args); err == nil {
			expression = args.Input
		} else {
			expression = task.Content[0].Parts[0].GetText()
		}
	}

	if expression == "" {
		return nil, fmt.Errorf("input expression is empty")
	}

	appLogger.Info(fmt.Sprintf("Processing expression: %s", expression))

	parts := strings.Fields(expression)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid expression format: %s", expression)
	}

	val1, err1 := strconv.ParseFloat(parts[0], 64)
	operator := parts[1]
	val2, err2 := strconv.ParseFloat(parts[2], 64)

	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("invalid numbers in expression: %s", expression)
	}

	var result float64
	switch operator {
	case "+":
		result = val1 + val2
	case "-":
		result = val1 - val2
	case "*":
		result = val1 * val2
	case "/":
		if val2 == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = val1 / val2
	default:
		return nil, fmt.Errorf("unsupported operator: %s", operator)
	}

	resultTask := &v1.AgentTask{
		TaskId:        uuid.New().String(),
		CorrelationId: task.CorrelationId,
		ParentTaskId:  task.TaskId,
		SourceAgentId: s.Metadata().Name,
		TargetAgentId: task.SourceAgentId,
		TaskName:      "Calculator Result",
		Content: []*v1.Content{
			{
				Parts: []*v1.Part{
					{Text: fmt.Sprintf("%f", result)},
				},
			},
		},
	}

	appLogger.Info(fmt.Sprintf("Calculation result: %f", result))
	return resultTask, nil
}
