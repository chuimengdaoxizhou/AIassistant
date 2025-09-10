package models

import (
	"time"
)

// AgentTask 是用于 Agent 之间通信和任务分配的核心结构。
type AgentTask struct {
	// --- 核心元数据 ---
	TaskID        string `json:"taskID"`                 // 当前任务的唯一标识符 (例如: UUID)。
	CorrelationID string `json:"correlationID"`          // 贯穿整个请求链的唯一 ID，由用户初次请求时生成。
	ParentTaskID  string `json:"parentTaskID,omitempty"` // 父任务ID，用于构建任务树。

	// --- 路由和命名 ---
	SourceAgentID string `json:"sourceAgentID"` // 发起任务的 Agent ID。
	TargetAgentID string `json:"targetAgentID"` // 目标 Agent ID 或角色。
	TaskName      string `json:"taskName"`      // 任务的名称，例如 "summarize_text", "generate_image"。

	// --- 载荷 (Payload) ---
	// Payload 包含了执行任务所需的具体数据。
	// 使用 json.RawMessage 可以灵活地传递任何结构，而无需在传输层解析它。
	Content Content `json:"content"`

	// --- 控制参数 ---
	CreatedAt      time.Time    `json:"createdAt"`                // 任务创建时间。
	TimeoutSeconds int          `json:"timeoutSeconds,omitempty"` // 任务超时（秒）。
	RetryPolicy    *RetryPolicy `json:"retryPolicy,omitempty"`    // 失败时的重试策略。
}

// RetryPolicy 定义了任务失败后的重试策略。
type RetryPolicy struct {
	MaxRetries   int     `json:"maxRetries"`   // 最大重试次数。
	BackoffCoeff float64 `json:"backoffCoeff"` // 退避系数 (例如: 2.0)。
	InitialDelay string  `json:"initialDelay"` // 初始延迟 (例如: "1s")。
}
