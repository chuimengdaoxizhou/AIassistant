package kafka

import (
	"Jarvis_2.0/backend/go/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"time"
)

const AgentLogTopic = "agent_logs"

// LogPublisher 封装了向 Kafka 发送任务日志的逻辑。
type LogPublisher struct {
	writer *kafka.Writer
}

// NewLogPublisher 创建一个新的 LogPublisher 实例。
func NewLogPublisher(client *KafkaClient) *LogPublisher {
	// 为日志主题创建一个新的 writer 实例配置
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      client.Config.Brokers,
		Topic:        AgentLogTopic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		BatchSize:    100,
	})
	return &LogPublisher{writer: writer}
}

// LogTaskProgress 将 TaskLogEntry 序列化为 JSON 并发送到 Kafka。
func (p *LogPublisher) LogTaskProgress(ctx context.Context, entry *models.TaskLogEntry) error {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(entry.TaskID),
		Value: jsonData,
	})

	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	return nil
}

// Close 关闭底层的 writer 连接。
func (p *LogPublisher) Close() error {
	return p.writer.Close()
}
