package models

import "time"

// TaskLogStatus 定义了任务日志的状态枚举。
type TaskLogStatus string

const (
	StatusThinking         TaskLogStatus = "THINKING"
	StatusCallingSubAgent  TaskLogStatus = "CALLING_SUB_AGENT"
	StatusCallingMCPTool   TaskLogStatus = "CALLING_MCP_TOOL" // 新增状态
	StatusObserving        TaskLogStatus = "OBSERVING"
	StatusFinished         TaskLogStatus = "FINISHED"
	StatusError            TaskLogStatus = "ERROR"
)

// TaskLogEntry 定义了发送到 Kafka 的任务进度日志的统一结构。
type TaskLogEntry struct {
	TaskID        string        `json:"task_id"`
	CorrelationID string        `json:"correlation_id"`
	Timestamp     time.Time     `json:"timestamp"`
	Status        TaskLogStatus `json:"status"`
	Message       string        `json:"message"`
	Content       interface{}   `json:"content,omitempty"`
}