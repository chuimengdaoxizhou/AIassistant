package publisher

import (
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
)

// ResultPublisher is responsible for publishing task results to Kafka.
type ResultPublisher struct {
	writer *kafka.Writer
	logger *logger.Logger
}

// NewResultPublisher creates a new ResultPublisher.
func NewResultPublisher(brokers []string, topic string, logger *logger.Logger) *ResultPublisher {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	return &ResultPublisher{
		writer: writer,
		logger: logger,
	}
}

// Publish sends a task result message to the Kafka topic.
func (p *ResultPublisher) Publish(ctx context.Context, key string, value interface{}) error {
	msgBytes, err := json.Marshal(value)
	if err != nil {
		p.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to marshal result for Kafka")
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: msgBytes,
	})
	if err != nil {
		p.logger.WithError(models.ErrorInfo{Message: err.Error()}).WithPayload(map[string]interface{}{"topic": p.writer.Topic}).Error("Failed to write result message to Kafka")
		return err
	}
	return nil
}

// Close closes the underlying Kafka writer.
func (p *ResultPublisher) Close() error {
	return p.writer.Close()
}
