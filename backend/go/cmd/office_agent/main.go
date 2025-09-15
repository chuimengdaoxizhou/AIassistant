package main

import (
	v1 "Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/internal/discovery/etcd"
	"Jarvis_2.0/backend/go/internal/office_agent/api"
	"Jarvis_2.0/backend/go/internal/office_agent/service"
	grpcserver "Jarvis_2.0/backend/go/pkg/grpc"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
)

const (
	ServiceName    = "office_agent"
	ServiceAddress = "localhost:9092" // 这个地址应该是可配置的
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
	appLogger := logger.New(ServiceName, "", "")
	appLogger.Info("Logger initialized")

	// 3. 初始化 etcd 服务发现客户端并注册服务
	sd, err := etcd.NewServiceDiscovery(cfg.Databases.Etcd.Endpoints)
	if err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to create service discovery client: %v", err))
	}
	stopChan, err := sd.Register(ServiceName, ServiceAddress, 10) // 10秒 TTL
	if err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to register service: %v", err))
	}
	defer close(stopChan) // 确保程序退出时停止心跳
	appLogger.Info(fmt.Sprintf("Service '%s' registered at '%s'", ServiceName, ServiceAddress))

	// 4. 准备 LLM 和 MinIO 配置
	llmConfig := service.LLMConfig{
		Provider: cfg.LLM.Provider,
		Model:    cfg.LLM.Gemini.Model,
		APIKey:   cfg.LLM.Gemini.APIKey,
	}
	minioConfig := cfg.Databases.MinIO

	// 5. 初始化服务核心逻辑 (它会自己创建 LLM 和 MinIO 客户端)
	ctx := context.Background()
	officeSvc, err := service.NewOfficeAgentService(ctx, llmConfig, minioConfig)
	if err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to create office agent service: %v", err))
	}
	defer officeSvc.Close() // 确保程序退出时关闭 MCP Host 连接

	officeGRPCHandler := api.NewOfficeAgentServerHandler(officeSvc)
	appLogger.Info("Service core and gRPC handler initialized")

	// 6. 初始化并启动 gRPC 服务器
	grpcServer, err := grpcserver.NewServer(cfg)
	if err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to create gRPC server: %v", err))
	}
	v1.RegisterAgentServiceServer(grpcServer.GetGRPCServer(), officeGRPCHandler)
	appLogger.Info("gRPC handler registered")

	appLogger.Info("Starting gRPC server on " + ServiceAddress)
	if err := grpcServer.ListenAndServe(); err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to start gRPC server: %v", err))
	}
}