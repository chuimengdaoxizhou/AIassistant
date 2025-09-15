package main

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/dal"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	ragv1 "Jarvis_2.0/api/proto/v1/rag"
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/internal/database/milvus"
	"Jarvis_2.0/backend/go/internal/database/mysql"
	"Jarvis_2.0/backend/go/internal/embedding"
	"Jarvis_2.0/backend/go/internal/llm"
	"Jarvis_2.0/backend/go/internal/rag_service/service"
	"Jarvis_2.0/backend/go/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	httpPort = ":8080"
	grpcPort = ":50051"
)

func main() {
	// 1. Initialize Logger
	// Note: Logger level should also come from config, hardcoded for now.
	logger.Init(logrus.InfoLevel)
	appLogger := logger.New("RAGService", "", "")
	appLogger.Info("Starting RAG Service...")

	// 2. Load Configuration
	// Assuming the config file is located at ./config/config.yaml relative to the executable
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	appLogger.Info("Configuration loaded successfully.")

	// 3. Initialize Dependencies
	db, err := mysql.GetDB(&cfg.Databases.MySQL)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	folderDal := dal.NewFolderDAL(db)

	milvusClient, err := milvus.GetClient(context.Background(), &cfg.Databases.Milvus)
	if err != nil {
		log.Fatalf("Failed to connect to Milvus: %v", err)
	}

	geminiEmbedding, err := embedding.NewGoogleModel(cfg.Embedding.Gemini.APIKey, cfg.Embedding.Gemini.Model)
	if err != nil {
		log.Fatalf("Failed to create Gemini embedding client: %v", err)
	}

	geminiLLM, err := llm.NewGemini(context.Background(), cfg.LLM.Gemini.Model, cfg.LLM.Gemini.APIKey, nil)
	if err != nil {
		log.Fatalf("Failed to create Gemini LLM client: %v", err)
	}

	// Get Cohere API key from environment as it's not in the config struct
	cohereAPIKey := os.Getenv("COHERE_API_KEY")
	if cohereAPIKey == "" {
		appLogger.Warn("COHERE_API_KEY environment variable not set. Reranker will not function.")
	}

	// 4. Create the RAG Service
	ragService := service.NewServer(*appLogger, folderDal, milvusClient, geminiEmbedding, geminiLLM, cfg.Databases.Milvus.Schema.CollectionName, cohereAPIKey)

	// 5. Start gRPC Server in a goroutine
	go func() {
		lis, err := net.Listen("tcp", grpcPort)
		if err != nil {
			log.Fatalf("Failed to listen for gRPC: %v", err)
		}
		grpcServer := grpc.NewServer()
		ragv1.RegisterRagServiceServer(grpcServer, ragService)
		appLogger.Info(fmt.Sprintf("gRPC server listening at %v", lis.Addr()))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// 6. Start Gin HTTP Server in a goroutine
	go func() {
		gin.SetMode(gin.ReleaseMode)
		router := gin.Default()
		httpHandler := NewHttpHandler(ragService)

		api := router.Group("/api/v1")
		{
			api.POST("/rag/query", httpHandler.query)
			api.POST("/rag/folders", httpHandler.createFolder)
			api.GET("/rag/folders", httpHandler.listFolders)
		}

		appLogger.Info(fmt.Sprintf("HTTP server listening at %s", httpPort))
		if err := router.Run(httpPort); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// 7. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Info("Shutting down servers...")

	// Add graceful shutdown logic for servers if needed
	appLogger.Info("Servers gracefully stopped")
}

// HttpHandler wraps the gRPC service to expose it via REST
type HttpHandler struct {
	service *service.Server
}

func NewHttpHandler(service *service.Server) *HttpHandler {
	return &HttpHandler{service: service}
}

func (h *HttpHandler) query(c *gin.Context) {
	var req ragv1.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Query(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HttpHandler) createFolder(c *gin.Context) {
	var req ragv1.CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.CreateFolder(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HttpHandler) listFolders(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	req := ragv1.ListFoldersRequest{UserId: userID}

	resp, err := h.service.ListFolders(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
