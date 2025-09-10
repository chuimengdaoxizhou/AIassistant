package models

// LogEntry 定义了用于结构化日志的统一数据格式。
// 这个结构旨在方便日志采集、传输，并最终在 Elasticsearch 中进行高效地解析、索引和分析。
type LogEntry struct {
	// ServiceName 是指产生这条日志的微服务或组件的名称。
	// 例如："backend-go", "vision-agent"
	ServiceName string `json:"service_name"`

	// TraceID 用于将跨越多个服务的单个请求串联起来，便于进行分布式追踪。
	TraceID string `json:"trace_id,omitempty"`

	// UserID 标识了与此日志事件相关的用户（如果适用）。
	UserID string `json:"user_id,omitempty"`

	// RequestInfo 包含了触发此日志的 HTTP 请求的详细信息。
	RequestInfo *RequestInfo `json:"request_info,omitempty"`

	// Error 包含了详细的错误信息，通常在日志级别为 Error 或更高时填充。
	Error *ErrorInfo `json:"error,omitempty"`

	// Payload 用于存放任何其他与业务逻辑相关的、需要记录的结构化数据。
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// RequestInfo 存储了关于 HTTP 请求的上下文信息。
type RequestInfo struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	RemoteAddr string `json:"remote_addr"`
	UserAgent  string `json:"user_agent"`
}

// ErrorInfo 存储了关于错误的结构化信息。
type ErrorInfo struct {
	Message    string `json:"message"`
	Stack      string `json:"stack,omitempty"`      // 错误的堆栈信息
	Type       string `json:"type,omitempty"`        // 错误的类型，例如 "database_error", "validation_error"
	StatusCode int    `json:"status_code,omitempty"` // 相关的HTTP状态码
}
