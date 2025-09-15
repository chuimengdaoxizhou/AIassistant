package api

import (
	"Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/office_agent/service"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
)

// OfficeAgentServerHandler 是 office_agent 的 gRPC 请求处理器。
type OfficeAgentServerHandler struct {
	v1.UnimplementedAgentServiceServer
	svc *service.OfficeAgentService
}

// NewOfficeAgentServerHandler 创建一个新的 gRPC 处理器实例。
func NewOfficeAgentServerHandler(svc *service.OfficeAgentService) *OfficeAgentServerHandler {
	return &OfficeAgentServerHandler{svc: svc}
}

// ExecuteTask 接收并执行一个 gRPC 任务请求。
func (h *OfficeAgentServerHandler) ExecuteTask(ctx context.Context, task *v1.AgentTask) (*v1.AgentTask, error) {
	return h.svc.Execute(ctx, task)
}

// GetMetadata 返回 agent 的元数据。
func (h *OfficeAgentServerHandler) GetMetadata(ctx context.Context, empty *emptypb.Empty) (*v1.AgentMetadata, error) {
	metadata := h.svc.Metadata()
	// 将内部的 agent.AgentMetadata 转换为 protobuf 的 v1.AgentMetadata
	return &v1.AgentMetadata{
		Name:              metadata.Name,
		Capability:        metadata.Capability,
		InputDescription:  metadata.InputDescription,
		OutputDescription: metadata.OutputDescription,
	}, nil
}
