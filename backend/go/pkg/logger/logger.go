package logger

import (
	"Jarvis_2.0/backend/go/internal/models"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger 是对 logrus 的封装，以提供更方便的结构化日志记录功能。
type Logger struct {
	entry *logrus.Entry
}

// Init 初始化全局的 logrus 配置。
// level: 设置日志级别 (e.g., logrus.InfoLevel, logrus.DebugLevel)。
func Init(level logrus.Level) {
	// 设置日志格式为 JSON，这对于后续的日志采集和分析至关重要。
	logrus.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// 设置日志输出到标准输出（终端）。
	logrus.SetOutput(os.Stdout)

	// 设置全局日志级别。
	logrus.SetLevel(level)
}

// New 创建一个新的 Logger 实例，并可以预设一些初始字段。
func New(serviceName, traceID, userID string) *Logger {
	return &Logger{
		entry: logrus.WithFields(logrus.Fields{
			"service_name": serviceName,
			"trace_id":     traceID,
			"user_id":      userID,
		}),
	}
}

// WithRequest 将请求信息添加到日志条目中。
func (l *Logger) WithRequest(req models.RequestInfo) *Logger {
	l.entry = l.entry.WithField("request_info", req)
	return l
}

// WithError 将错误信息添加到日志条目中。
func (l *Logger) WithError(err models.ErrorInfo) *Logger {
	l.entry = l.entry.WithField("error", err)
	return l
}

// WithPayload 将自定义的业务数据添加到日志条目中。
func (l *Logger) WithPayload(payload map[string]interface{}) *Logger {
	l.entry = l.entry.WithField("payload", payload)
	return l
}

// Info 记录一条信息级别的日志。
func (l *Logger) Info(message string) {
	l.entry.Info(message)
}

// Warn 记录一条警告级别的日志。
func (l *Logger) Warn(message string) {
	l.entry.Warn(message)
}

// Error 记录一条错误级别的日志。
func (l *Logger) Error(message string) {
	l.entry.Error(message)
}

// Debug 记录一条调试级别的日志。
func (l *Logger) Debug(message string) {
	l.entry.Debug(message)
}

// Fatal 记录一条致命错误级别的日志，并终止程序。
func (l *Logger) Fatal(message string) {
	l.entry.Fatal(message)
}
