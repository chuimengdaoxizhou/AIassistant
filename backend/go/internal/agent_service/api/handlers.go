package api

import (
	v1 "Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/agent_service/service"
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AgentServerHandler 是 agent_service 的 gRPC 处理器。
type AgentServerHandler struct {
	// 必须嵌入 UnimplementedAgentServiceServer 以满足接口要求。
	v1.UnimplementedAgentServiceServer
	svc *service.AgentService
}

// NewAgentServerHandler 创建一个新的 gRPC 处理器。
func NewAgentServerHandler(svc *service.AgentService) *AgentServerHandler {
	return &AgentServerHandler{
		svc: svc,
	}
}

// ExecuteTask 接收 gRPC 请求并将其委托给 AgentService。
func (h *AgentServerHandler) ExecuteTask(ctx context.Context, task *v1.AgentTask) (*v1.AgentTask, error) {
	resultTask, err := h.svc.Execute(ctx, task)
	if err != nil {
		// 检查是否是 MCPError，如果是，则返回包含 FunctionCall 的 AgentTask
		var mcpErr *service.MCPError
		if errors.As(err, &mcpErr) {
			return mcpErr.Task, nil // 返回包含 MCP FunctionCall 的 AgentTask
		}
		// 其他错误则返回标准的 gRPC 错误
		return nil, status.Errorf(codes.Internal, "internal error: %v", err)
	}
	return resultTask, nil
}
