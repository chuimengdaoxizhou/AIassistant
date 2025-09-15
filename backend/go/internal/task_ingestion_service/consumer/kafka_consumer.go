package consumer

import (
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"github.com/segmentio/kafka-go"
)

// ResultConsumer is responsible for consuming task results from Kafka.
type ResultConsumer struct {
	reader *kafka.Reader
	logger *logger.Logger
}

// NewResultConsumer creates a new ResultConsumer.
func NewResultConsumer(brokers []string, topic, groupID string, logger *logger.Logger) *ResultConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	return &ResultConsumer{
		reader: reader,
		logger: logger,
	}
}

// Start begins consuming messages from the Kafka topic.
func (c *ResultConsumer) Start(ctx context.Context, handler func(kafka.Message) error) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("Stopping Kafka result consumer...")
				return
			default:
				msg, err := c.reader.FetchMessage(ctx)
				if err != nil {
					if ctx.Err() == nil {
						c.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Error fetching message from Kafka")
					}
					continue
				}

				if err := handler(msg); err != nil {
					c.logger.WithError(models.ErrorInfo{Message: err.Error()}).WithPayload(map[string]interface{}{
						"topic":     msg.Topic,
						"partition": msg.Partition,
						"offset":    msg.Offset,
					}).Error("Error handling Kafka message")
				}

				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					c.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to commit Kafka message")
				}
			}
		}
	}()
}

// Close closes the underlying Kafka reader.
func (c *ResultConsumer) Close() error {
	return c.reader.Close()
}
