package api

import (
	v1 "Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/calculator_agent/service"
	"context"
	"github.com/golang/protobuf/ptypes/empty"
)

// CalculatorServerHandler 是 calculator_agent 的 gRPC 处理器。
type CalculatorServerHandler struct {
	v1.UnimplementedAgentServiceServer
	svc *service.CalculatorService
}

// NewCalculatorServerHandler 创建一个新的 gRPC 处理器。
func NewCalculatorServerHandler(svc *service.CalculatorService) *CalculatorServerHandler {
	return &CalculatorServerHandler{
		svc: svc,
	}
}

// ExecuteTask 接收 gRPC 请求并将其委托给 CalculatorService。
func (h *CalculatorServerHandler) ExecuteTask(ctx context.Context, task *v1.AgentTask) (*v1.AgentTask, error) {
	return h.svc.Execute(ctx, task)
}

// GetMetadata 接收 gRPC 请求并返回 Agent 的元数据。
func (h *CalculatorServerHandler) GetMetadata(ctx context.Context, _ *empty.Empty) (*v1.AgentMetadata, error) {
	meta := h.svc.Metadata()
	return &v1.AgentMetadata{
		Name:              meta.Name,
		Capability:        meta.Capability,
		InputDescription:  meta.InputDescription,
		OutputDescription: meta.OutputDescription,
	}, nil
}