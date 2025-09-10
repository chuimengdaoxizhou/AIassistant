package consumer

import (
	"Jarvis_2.0/backend/go/internal/database/kafka"
	"Jarvis_2.0/backend/go/internal/memory/service"
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"encoding/json"
)

// KafkaConsumer consumes messages from a Kafka topic and processes them with the MemoryService.
type KafkaConsumer struct {
	kafkaClient   *kafka.KafkaClient
	memoryService *service.MemoryService
	logger        *logger.Logger
}

// NewKafkaConsumer creates a new KafkaConsumer.
func NewKafkaConsumer(kafkaClient *kafka.KafkaClient, memoryService *service.MemoryService, logger *logger.Logger) *KafkaConsumer {
	return &KafkaConsumer{
		kafkaClient:   kafkaClient,
		memoryService: memoryService,
		logger:        logger,
	}
}

// Start starts the Kafka consumer.
func (c *KafkaConsumer) Start(ctx context.Context) {
	go func() {
		for {
			msg, err := c.kafkaClient.Reader.FetchMessage(ctx)
			if err != nil {
				c.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("failed to fetch message")
				continue
			}

			var historyContent models.HistoryContent
			if err := json.Unmarshal(msg.Value, &historyContent); err != nil {
				c.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("failed to unmarshal message")
				continue
			}

			if err := c.memoryService.AddMemory(ctx, &historyContent); err != nil {
				c.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("failed to add memory")
				continue
			}

			if err := c.kafkaClient.Reader.CommitMessages(ctx, msg); err != nil {
				c.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("failed to commit message")
			}
		}
	}()
}
