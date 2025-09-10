package agent

import (
	"Jarvis_2.0/api/proto/v1"
	"context"
)

// Agent 定义了系统中所有 Agent（包括 MainAgent 和 SubAgent）必须实现的接口。
type Agent interface {
	// Metadata 返回 Agent 的能力描述。
	Metadata() AgentMetadata
	// Execute 执行任务。
	Execute(ctx context.Context, task *v1.AgentTask) (*v1.AgentTask, error)
}
