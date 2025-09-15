package api

import (
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/internal/task_ingestion_service/service"
	"Jarvis_2.0/backend/go/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"strconv"
)

// API provides handlers for the task ingestion service.
type API struct {
	service *service.TaskService
	logger  *logger.Logger
	upgrader websocket.Upgrader
}

// NewAPI creates a new API handler.
func NewAPI(service *service.TaskService, logger *logger.Logger) *API {
	return &API{
		service: service,
		logger:  logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // In production, implement a proper origin check.
			},
		},
	}
}

// SubmitTaskHandler handles the submission of a new task.
func (a *API) SubmitTaskHandler(c *gin.Context) {
	userID, _ := c.Get("userID")

	var payload struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		a.logger.WithError(models.ErrorInfo{Message: err.Error()}).Warn("Invalid request payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	task, err := a.service.SubmitTask(c.Request.Context(), userID.(string), payload)
	if err != nil {
		// The service layer already logged the detailed error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit task"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID})
}

// GetTaskHandler handles requests to get a single task by its ID.
func (a *API) GetTaskHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	taskID := c.Param("id")

	task, err := a.service.GetTaskByID(c.Request.Context(), taskID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found or not authorized"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// GetTasksHandler handles requests to get a list of tasks for the user.
func (a *API) GetTasksHandler(c *gin.Context) {
	userID, _ := c.Get("userID")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	tasks, err := a.service.GetUserTasks(c.Request.Context(), userID.(string), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// WebSocketHandler handles WebSocket connection upgrades.
func (a *API) WebSocketHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	
	conn, err := a.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		a.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to upgrade WebSocket connection")
		return
	}

	a.service.AddConnection(userID.(string), conn)

	conn.SetCloseHandler(func(code int, text string) error {
		a.service.RemoveConnection(userID.(string))
		return nil
	})
	
	go func() {
		defer a.service.RemoveConnection(userID.(string))
		for {
			if _, _, err := conn.NextReader(); err != nil {
				break
			}
		}
	}()
}
