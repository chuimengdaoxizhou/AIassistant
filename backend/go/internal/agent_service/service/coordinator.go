package service

import (
	"Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/agent_service/publisher"
	"Jarvis_2.0/backend/go/internal/agent_service/store"
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
)

// Coordinator orchestrates the process of consuming tasks, executing them, and publishing results.
type Coordinator struct {
	agentService     *AgentService
	resultPublisher  *publisher.ResultPublisher
	taskUpdater      store.TaskUpdater
	contentProcessor *store.ContentProcessor // Added ContentProcessor dependency
	logger           *logger.Logger
}

// NewCoordinator creates a new Coordinator.
func NewCoordinator(agentService *AgentService, publisher *publisher.ResultPublisher, updater store.TaskUpdater, processor *store.ContentProcessor, logger *logger.Logger) *Coordinator {
	return &Coordinator{
		agentService:     agentService,
		resultPublisher:  publisher,
		taskUpdater:      updater,
		contentProcessor: processor, // Injected ContentProcessor
		logger:           logger,
	}
}

// ProcessTask is the handler for each Kafka message.
func (c *Coordinator) ProcessTask(ctx context.Context, msg kafka.Message) error {
	var task models.TaskRecord
	if err := json.Unmarshal(msg.Value, &task); err != nil {
		c.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to unmarshal task from Kafka")
		return err
	}

	taskLogger := logger.New("AgentCoordinator", task.ID, task.UserID)
	taskLogger.Info("Starting to process task")

	payload, ok := task.Payload.(map[string]interface{})
	content, contentOK := payload["content"].(string)
	if !ok || !contentOK || content == "" {
		errMsg := "Invalid or empty task payload content"
		taskLogger.Warn(errMsg)
		_ = c.updateAndPublishResult(ctx, &task, models.TaskStatusFailed, errMsg)
		return nil
	}

	protoTask := &v1.AgentTask{
		TaskId:        task.ID,
		CorrelationId: task.ID,
		Content: []*v1.Content{
			{
				Role:  string(models.SpeakerUser),
				Parts: []*v1.Part{{Text: content}},
			},
		},
	}

	resultTask, err := c.agentService.RunReActLoop(ctx, protoTask)

	if err != nil {
		taskLogger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Task execution failed")
		return c.updateAndPublishResult(ctx, &task, models.TaskStatusFailed, err.Error())
	}

	taskLogger.Info("Task execution successful")

	finalContent := models.ConvertProtoToModelsContent(resultTask.GetContent())

	return c.updateAndPublishResult(ctx, &task, models.TaskStatusSuccess, finalContent)
}

func (c *Coordinator) updateAndPublishResult(ctx context.Context, task *models.TaskRecord, status models.TaskStatus, resultData interface{}) error {
	var finalResultData interface{} = resultData
	var err error

	// If the task was successful, process the content for storage (e.g., upload files to MinIO).
	if status == models.TaskStatusSuccess {
		contentToProcess, ok := resultData.([]models.Content)
		if ok {
			finalResultData, err = c.contentProcessor.ProcessAndStoreContent(ctx, contentToProcess)
			if err != nil {
				c.logger.WithError(models.ErrorInfo{Message: err.Error()}).WithPayload(map[string]interface{}{"taskID": task.ID}).Error("Failed to process content for storage")
				// If processing fails, we should probably fail the task.
				status = models.TaskStatusFailed
				finalResultData = "Failed to process and store multimodal content"
			}
		}
	}

	if err := c.taskUpdater.UpdateTaskResult(ctx, task.ID, status, finalResultData); err != nil {
		c.logger.WithError(models.ErrorInfo{Message: err.Error()}).WithPayload(map[string]interface{}{"taskID": task.ID}).Error("Failed to update task result in MongoDB")
	}

	resultRecord := models.TaskRecord{
		ID:     task.ID,
		UserID: task.UserID,
		Status: status,
	}
	if status == models.TaskStatusSuccess {
		resultRecord.Result = finalResultData
	} else {
		resultRecord.Error, _ = finalResultData.(string)
	}

	if err := c.resultPublisher.Publish(ctx, task.ID, resultRecord); err != nil {
		c.logger.WithError(models.ErrorInfo{Message: err.Error()}).WithPayload(map[string]interface{}{"taskID": task.ID}).Error("Failed to publish task result to Kafka")
		return err
	}

	c.logger.WithPayload(map[string]interface{}{"taskID": task.ID, "status": status}).Info("Successfully updated and published task result")
	return nil
}
