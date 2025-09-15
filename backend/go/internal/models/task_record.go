package models

import (
	"time"
)

// TaskStatus 定义了任务的几种可能状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSuccess   TaskStatus = "success"
	TaskStatusFailed    TaskStatus = "failed"
)

// TaskRecord 代表一个持久化的 agent 任务记录
type TaskRecord struct {
	ID          string      `bson:"_id"`         // 任务唯一ID (使用 a UUID string)
	UserID      string      `bson:"user_id"`     // 提交任务的用户ID
	Status      TaskStatus  `bson:"status"`      // 任务当前状态
	Payload     interface{} `bson:"payload"`     // 任务的输入/内容
	Result      interface{} `bson:"result"`      // 任务成功后的输出结果
	Error       string      `bson:"error"`       // 任务失败时的错误信息
	SubmittedAt time.Time   `bson:"submitted_at"`// 任务提交时间
	CompletedAt time.Time   `bson:"completed_at"`// 任务完成时间
}
