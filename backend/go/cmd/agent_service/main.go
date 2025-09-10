package main

import (
	"Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/agent"
	"Jarvis_2.0/backend/go/internal/agent_service/api"
	"Jarvis_2.0/backend/go/internal/agent_service/service"
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/internal/database/kafka"
	"Jarvis_2.0/backend/go/internal/discovery/etcd"
	"Jarvis_2.0/backend/go/internal/llm"
	grpcserver "Jarvis_2.0/backend/go/pkg/grpc"
	"Jarvis_2.0/backend/go/pkg/logger"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// 2. 初始化 Logger
	level, err := logrus.ParseLevel(cfg.Logger.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.Init(level)
	appLogger := logger.New("agent_service", "", "")
	appLogger.Info("Logger initialized for Agent Service")

	// 3. 初始化 Kafka 客户端和日志发布器
	kafkaClient, err := kafka.GetClient(&cfg.Databases.Kafka)
	if err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to create kafka client: %v", err))
	}
	logPublisher := kafka.NewLogPublisher(kafkaClient)
	defer func(logPublisher *kafka.LogPublisher) {
		err := logPublisher.Close()
		if err != nil {
			appLogger.Error(fmt.Sprintf("Failed to close log publisher cleanly: %v", err))
		}
	}(logPublisher)
	appLogger.Info("Kafka log publisher initialized")

	// 4. 初始化 etcd 服务发现客户端
	sd, err := etcd.NewServiceDiscovery(cfg.Databases.Etcd.Endpoints)
	if err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to create service discovery client: %v", err))
	}

	// 5. 初始化 DistributedRegistry 并发现子 Agent
	distributedRegistry := agent.NewDistributedRegistry(sd)
	if err := distributedRegistry.DiscoverAndCacheAgents(); err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to discover agents: %v", err))
	}
	appLogger.Info("Distributed agent registry initialized and agents discovered")

	// 6. 初始化 LLM 客户端 (以 Gemini 为例)
	llmClient, err := llm.NewLLM("gemini", cfg.LLM.Gemini.Model, cfg.LLM.Gemini.APIKey, "", nil)
	if err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to create LLM client: %v", err))
	}
	appLogger.Info("LLM client initialized")

	// 7. 初始化 AgentService
	agentSvc := service.NewAgentService(llmClient, distributedRegistry, logPublisher, nil)
	appLogger.Info("Agent service core initialized")

	// 8. 初始化 gRPC 服务器
	grpcServer, err := grpcserver.NewServer(cfg)
	if err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to create gRPC server: %v", err))
	}

	// 9. 注册 gRPC 处理器
	agentGRPCHandler := api.NewAgentServerHandler(agentSvc)
	v1.RegisterAgentServiceServer(grpcServer.GetGRPCServer(), agentGRPCHandler)
	appLogger.Info("gRPC handler registered")

	// 10. 启动 gRPC 服务器
	// TODO: Get port from config
	grpcPort := ":9090"
	appLogger.Info("Starting gRPC server on port " + grpcPort)
	if err := grpcServer.ListenAndServe(); err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to start gRPC server: %v", err))
	}
}
