package publisher

import (
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
)

// TaskPublisher is responsible for publishing tasks to Kafka.
type TaskPublisher struct {
	writer *kafka.Writer
	logger *logger.Logger
}

// NewTaskPublisher creates a new TaskPublisher.
func NewTaskPublisher(brokers []string, topic string, logger *logger.Logger) *TaskPublisher {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	return &TaskPublisher{
		writer: writer,
		logger: logger,
	}
}

// Publish sends a task message to the Kafka topic.
func (p *TaskPublisher) Publish(ctx context.Context, key string, value interface{}) error {
	msgBytes, err := json.Marshal(value)
	if err != nil {
		p.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to marshal task for Kafka")
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: msgBytes,
	})
	if err != nil {
		p.logger.WithError(models.ErrorInfo{Message: err.Error()}).WithPayload(map[string]interface{}{"topic": p.writer.Topic}).Error("Failed to write message to Kafka")
		return err
	}
	return nil
}

// Close closes the underlying Kafka writer.
func (p *TaskPublisher) Close() error {
	return p.writer.Close()
}
