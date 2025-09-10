package main

import (
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/internal/database/mysql"
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/internal/user_service/api"
	"Jarvis_2.0/backend/go/internal/user_service/service"
	"Jarvis_2.0/backend/go/internal/user_service/store"
	"Jarvis_2.0/backend/go/pkg/logger"
	"log"

	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Initialize logger
	level, err := logrus.ParseLevel(cfg.Logger.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.Init(level)
	appLogger := logger.New("user_service", "", "")

	appLogger.Info("Logger initialized")

	// Initialize database connection
	db, err := mysql.GetDB(&cfg.Databases.MySQL)
	if err != nil {
		appLogger.Fatal(err.Error())
	}
	appLogger.Info("Database connection established")

	// Auto-migrate database schema
	err = db.AutoMigrate(&models.User{}, &models.AuthRole{}, &models.Permission{})
	if err != nil {
		appLogger.Fatal(err.Error())
	}
	appLogger.Info("Database migration completed")

	// Initialize dependencies (Store -> Service -> Handler)
	userStore := store.NewStore(db)
	userService := service.NewService(userStore, cfg.Auth.JwtSecret)
	apiHandler := api.NewHandler(userService)
	appLogger.Info("Dependencies injected")

	// Setup and start Gin router
	router := api.SetupRouter(apiHandler, cfg.Auth.JwtSecret)
	appLogger.Info("Router setup completed")

	serverAddress := ":8080" // TODO: Get port from config
	appLogger.Info("Starting server on " + serverAddress)

	if err := router.Run(serverAddress); err != nil {
		appLogger.Fatal(err.Error())
	}
}