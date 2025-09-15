package main

import (
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/internal/database/mongo"
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/internal/task_ingestion_service/api"
	"Jarvis_2.0/backend/go/internal/task_ingestion_service/consumer"
	"Jarvis_2.0/backend/go/internal/task_ingestion_service/publisher"
	"Jarvis_2.0/backend/go/internal/task_ingestion_service/service"
	"Jarvis_2.0/backend/go/internal/task_ingestion_service/store"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("backend/go/internal/config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logLevel, err := logrus.ParseLevel(cfg.Logger.Level)
	if err != nil {
		log.Fatalf("Invalid logger level: %v", err)
	}
	logger.Init(logLevel)
	
	// Create a single base logger for the service
	serviceLogger := logger.New("TaskIngestionService", "", "")

	// Connect to MongoDB using the singleton GetClient
	mongoClient, err := mongo.GetClient(&cfg.Databases.MongoDB)
	if err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Fatal("Failed to connect to MongoDB")
	}
	db := mongoClient.Database(cfg.Databases.MongoDB.Database)
	serviceLogger.Info("Successfully connected to MongoDB")

	// Create components with logger injection
	taskStore := store.NewMongoTaskStore(db, cfg.TaskIngestion.MongoCollection)
	connManager := service.NewConnectionManager()
	taskPublisher := publisher.NewTaskPublisher(cfg.Databases.Kafka.Brokers, cfg.TaskIngestion.KafkaTasksTopic, serviceLogger)
	taskService := service.NewTaskService(taskStore, connManager, taskPublisher, serviceLogger)
	resultConsumer := consumer.NewResultConsumer(cfg.Databases.Kafka.Brokers, cfg.TaskIngestion.KafkaResultsTopic, "task-ingestion-group", serviceLogger)

	// Start Kafka consumer
	ctx, cancel := context.WithCancel(context.Background())
	resultConsumer.Start(ctx, taskService.HandleResult)
	serviceLogger.Info("Kafka result consumer started")

	// Setup HTTP server
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	apiHandler := api.NewAPI(taskService, serviceLogger)
	api.RegisterRoutes(router, apiHandler)

	srv := &http.Server{
		Addr:    cfg.TaskIngestion.ServerAddress,
		Handler: router,
	}

	// Start server
	go func() {
		serviceLogger.Info("Starting HTTP server on " + srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Fatal("HTTP server failed to start")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	serviceLogger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Fatal("Server forced to shutdown")
	}

	cancel()
	if err := taskPublisher.Close(); err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Error closing Kafka publisher")
	}
	if err := resultConsumer.Close(); err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Error closing Kafka consumer")
	}
	// Use the provided Close function for MongoDB
	if err := mongo.Close(context.Background()); err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Error disconnecting from MongoDB")
	}

	serviceLogger.Info("Server gracefully stopped")
}