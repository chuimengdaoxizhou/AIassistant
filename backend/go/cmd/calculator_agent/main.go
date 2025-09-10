package main

import (
	v1 "Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/calculator_agent/api"
	"Jarvis_2.0/backend/go/internal/calculator_agent/service"
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/internal/discovery/etcd"
	grpcserver "Jarvis_2.0/backend/go/pkg/grpc"
	"Jarvis_2.0/backend/go/pkg/logger"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
)

const (
	ServiceName    = "calculator_agent"
	ServiceAddress = "localhost:9091" // 这个地址应该是可配置的
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

	// 4. 初始化服务核心逻辑和 gRPC 处理器
	calcSvc := service.NewCalculatorService()
	calcGRPCHandler := api.NewCalculatorServerHandler(calcSvc)
	appLogger.Info("Service core and gRPC handler initialized")

	// 5. 初始化并启动 gRPC 服务器
	grpcServer, err := grpcserver.NewServer(cfg)
	if err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to create gRPC server: %v", err))
	}
	v1.RegisterAgentServiceServer(grpcServer.GetGRPCServer(), calcGRPCHandler)
	appLogger.Info("gRPC handler registered")

	appLogger.Info("Starting gRPC server on " + ServiceAddress)
	if err := grpcServer.ListenAndServe(); err != nil {
		appLogger.Fatal(fmt.Sprintf("Failed to start gRPC server: %v", err))
	}
}
