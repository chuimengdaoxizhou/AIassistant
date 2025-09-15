package consumer

import (
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"github.com/segmentio/kafka-go"
)

// TaskConsumer is responsible for consuming tasks from Kafka.
type TaskConsumer struct {
	reader *kafka.Reader
	logger *logger.Logger
}

// NewTaskConsumer creates a new TaskConsumer.
func NewTaskConsumer(brokers []string, topic, groupID string, logger *logger.Logger) (*TaskConsumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	return &TaskConsumer{reader: reader, logger: logger}, nil
}

// Start begins consuming messages from the Kafka topic.
func (c *TaskConsumer) Start(ctx context.Context, handler func(context.Context, kafka.Message) error) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("Stopping Kafka task consumer...")
				return
			default:
				msg, err := c.reader.FetchMessage(ctx)
				if err != nil {
					if ctx.Err() == nil {
						c.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Error fetching message from Kafka")
					}
					continue
				}

				if err := handler(ctx, msg); err != nil {
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
func (c *TaskConsumer) Close() error {
	return c.reader.Close()
}
