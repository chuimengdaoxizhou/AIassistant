package service

import (
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/internal/task_ingestion_service/store"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"time"
)

// TaskService provides core business logic for task ingestion and result handling.
type TaskService struct {
	store       store.TaskStore
	connManager *ConnectionManager
	publisher   TaskPublisher
	logger      *logger.Logger
}

// TaskPublisher defines the interface for publishing tasks.
type TaskPublisher interface {
	Publish(ctx context.Context, key string, value interface{}) error
	Close() error
}

// NewTaskService creates a new TaskService.
func NewTaskService(store store.TaskStore, connManager *ConnectionManager, publisher TaskPublisher, logger *logger.Logger) *TaskService {
	return &TaskService{
		store:       store,
		connManager: connManager,
		publisher:   publisher,
		logger:      logger,
	}
}

// AddConnection adds a new WebSocket connection for a user.
func (s *TaskService) AddConnection(userID string, conn *websocket.Conn) {
	s.connManager.Add(userID, conn)
	s.logger.Info("WebSocket connection added for user: " + userID)
}

// RemoveConnection removes a WebSocket connection for a user.
func (s *TaskService) RemoveConnection(userID string) {
	s.connManager.Remove(userID)
	s.logger.Info("WebSocket connection removed for user: " + userID)
}

// SubmitTask creates a new task, stores it, and publishes it to Kafka.
func (s *TaskService) SubmitTask(ctx context.Context, userID string, payload interface{}) (*models.TaskRecord, error) {
	task := &models.TaskRecord{
		ID:          uuid.New().String(),
		UserID:      userID,
		Status:      models.TaskStatusPending,
		Payload:     payload,
		SubmittedAt: time.Now(),
	}

	if err := s.store.Create(ctx, task); err != nil {
		s.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to create task in store")
		return nil, err
	}

	if err := s.publisher.Publish(ctx, task.ID, task); err != nil {
		s.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to publish task to Kafka")
		task.Status = models.TaskStatusFailed
		task.Error = "Failed to publish to Kafka"
		task.CompletedAt = time.Now()
		_ = s.store.Update(ctx, task)
		return nil, err
	}

	return task, nil
}

// HandleResult processes a task result received from Kafka.
func (s *TaskService) HandleResult(msg kafka.Message) error {
	var resultTask models.TaskRecord
	if err := json.Unmarshal(msg.Value, &resultTask); err != nil {
		s.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to unmarshal task result from Kafka")
		return err
	}

	task, err := s.store.GetByID(context.Background(), resultTask.ID)
	if err != nil {
		s.logger.WithError(models.ErrorInfo{Message: err.Error()}).WithPayload(map[string]interface{}{"taskID": resultTask.ID}).Error("Error getting task by ID")
		return err
	}
	if task == nil {
		s.logger.WithPayload(map[string]interface{}{"taskID": resultTask.ID}).Warn("Received result for unknown task ID")
		return nil
	}

	task.Status = resultTask.Status
	task.Result = resultTask.Result
	task.Error = resultTask.Error
	task.CompletedAt = time.Now()

	if err := s.store.Update(context.Background(), task); err != nil {
		s.logger.WithError(models.ErrorInfo{Message: err.Error()}).WithPayload(map[string]interface{}{"taskID": task.ID}).Error("Failed to update task in store")
		return err
	}

	s.connManager.SendMessage(task.UserID, msg.Value)
	return nil
}

// GetTaskByID retrieves a single task by its ID for a specific user.
func (s *TaskService) GetTaskByID(ctx context.Context, taskID, userID string) (*models.TaskRecord, error) {
	task, err := s.store.GetByID(ctx, taskID)
	if err != nil {
		s.logger.WithError(models.ErrorInfo{Message: err.Error()}).WithPayload(map[string]interface{}{"taskID": taskID}).Error("Failed to get task by ID from store")
		return nil, err
	}
	if task != nil && task.UserID != userID {
		s.logger.WithPayload(map[string]interface{}{"taskID": taskID, "requestingUserID": userID}).Warn("User attempted to access unauthorized task")
		return nil, nil
	}
	return task, nil
}

// GetUserTasks retrieves all tasks for a specific user with pagination.
func (s *TaskService) GetUserTasks(ctx context.Context, userID string, page, limit int) ([]*models.TaskRecord, error) {
	tasks, err := s.store.GetByUserID(ctx, userID, page, limit)
	if err != nil {
		s.logger.WithError(models.ErrorInfo{Message: err.Error()}).WithPayload(map[string]interface{}{"userID": userID}).Error("Failed to get user tasks from store")
		return nil, err
	}
	return tasks, nil
}
