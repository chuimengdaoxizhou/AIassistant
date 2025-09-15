package main

import (
	"Jarvis_2.0/backend/go/internal/agent"
	"Jarvis_2.0/backend/go/internal/agent_service/consumer"
	"Jarvis_2.0/backend/go/internal/agent_service/publisher"
	"Jarvis_2.0/backend/go/internal/agent_service/service"
	"Jarvis_2.0/backend/go/internal/agent_service/store"
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/internal/database/kafka"
	"Jarvis_2.0/backend/go/internal/database/minio"
	"Jarvis_2.0/backend/go/internal/database/mongo"
	"Jarvis_2.0/backend/go/internal/discovery/etcd"
	"Jarvis_2.0/backend/go/internal/llm"
	"Jarvis_2.0/backend/go/internal/mcp"
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := config.LoadConfig("backend/go/internal/config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	logLevel, err := logrus.ParseLevel(cfg.Logger.Level)
	if err != nil {
		log.Fatalf("Invalid logger level: %v", err)
	}
	logger.Init(logLevel)
	serviceLogger := logger.New("AgentService", "", "")

	// --- 1. 初始化服务发现并发现所有子Agent ---
	sd, err := etcd.NewServiceDiscovery(cfg.Databases.Etcd.Endpoints)
	if err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Fatal("Failed to create service discovery client")
	}
	defer sd.Close()

	registry := agent.NewDistributedRegistry(sd)
	if err := registry.DiscoverAndCacheAgents(); err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Warn("Failed to discover agents from etcd, proceeding with limited capabilities.")
	}

	// --- 2. 准备工具并初始化LLM客户端 ---
	// 获取所有工具的元数据 (统一为 proto 类型)
	subAgentProtoMetadata := registry.GetAgentMetadata()
	mcpToolsProtoMetadata := mcp.GetTools()
	allToolsProtoMetadata := append(subAgentProtoMetadata, mcpToolsProtoMetadata...)

	// 将所有元数据一次性转换为LLM的FunctionDeclaration
	toolDeclarations := models.ConvertAgentMetadataToFunctionDeclarations(allToolsProtoMetadata)

	serviceLogger.Info(fmt.Sprintf("Loaded %d tools for the LLM: %d sub-agents and %d MCP tools.", len(allToolsProtoMetadata), len(subAgentProtoMetadata), len(mcpToolsProtoMetadata)))

	llmClient, err := llm.NewClient(cfg.LLM, toolDeclarations)
	if err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Fatal("Failed to create LLM client")
	}

	// --- 3. 初始化其他服务依赖 ---
	kafkaClient, err := kafka.GetClient(&cfg.Databases.Kafka)
	if err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Fatal("Failed to create Kafka client")
	}
	defer kafkaClient.Close()

	logPublisher := kafka.NewLogPublisher(kafkaClient)

	mongoClient, err := mongo.GetClient(&cfg.Databases.MongoDB)
	if err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Fatal("Failed to connect to MongoDB")
	}
	db := mongoClient.Database(cfg.Databases.MongoDB.Database)

	minioClient, err := minio.GetClient(&cfg.Databases.MinIO)
	if err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Fatal("Failed to connect to MinIO")
	}

	taskUpdater := store.NewMongoTaskUpdater(db, cfg.TaskIngestion.MongoCollection)
	contentProcessor := store.NewContentProcessor(minioClient, serviceLogger)

	// --- 4. 初始化核心服务和协调器 ---
	agentService := service.NewAgentService(llmClient, registry, logPublisher, nil)
	resultPublisher := publisher.NewResultPublisher(cfg.Databases.Kafka.Brokers, cfg.TaskIngestion.KafkaResultsTopic, serviceLogger)
	coordinator := service.NewCoordinator(agentService, resultPublisher, taskUpdater, contentProcessor, serviceLogger)

	// --- 5. 启动Kafka消费者 ---
	taskConsumer, err := consumer.NewTaskConsumer(cfg.Databases.Kafka.Brokers, cfg.TaskIngestion.KafkaTasksTopic, "agent-service-group", serviceLogger)
	if err != nil {
		serviceLogger.WithError(models.ErrorInfo{Message: err.Error()}).Fatal("Failed to create Kafka task consumer")
	}

	ctx, cancel := context.WithCancel(context.Background())
	taskConsumer.Start(ctx, coordinator.ProcessTask)
	serviceLogger.Info("Agent service coordinator started")

	// --- 6. 优雅关停 ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	serviceLogger.Info("Shutting down agent service...")
	cancel()
	logPublisher.Close()
	resultPublisher.Close()
	taskConsumer.Close()
	mongo.Close(context.Background())
	minio.Close()
	serviceLogger.Info("Agent service gracefully stopped")
}