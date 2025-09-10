package main

import (
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/internal/database/kafka"
	"Jarvis_2.0/backend/go/internal/database/milvus"
	"Jarvis_2.0/backend/go/internal/database/neo4j"
	"Jarvis_2.0/backend/go/internal/embedding"
	"Jarvis_2.0/backend/go/internal/llm"
	"Jarvis_2.0/backend/go/internal/memory/consumer"
	"Jarvis_2.0/backend/go/internal/memory/extractor"
	"Jarvis_2.0/backend/go/internal/memory/service"
	"Jarvis_2.0/backend/go/internal/memory/store"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	appLogger := logger.New("memory_service", "", "")

	// Initialize database clients
	ctx := context.Background()
	milvusClient, err := milvus.GetClient(ctx, &cfg.Databases.Milvus)
	if err != nil {
		appLogger.Fatal(err.Error())
	}
	defer milvusClient.Close()

	neo4jClient, err := neo4j.GetClient(ctx, &cfg.Databases.Neo4j)
	if err != nil {
		appLogger.Fatal(err.Error())
	}
	defer neo4jClient.Close(ctx)

	kafkaClient, err := kafka.GetClient(&cfg.Databases.Kafka)
	if err != nil {
		appLogger.Fatal(err.Error())
	}
	defer kafka.Close()

	// Initialize embedding and LLM clients
	embedder, err := embedding.NewEmdModel(cfg.Embedding.Provider, cfg.Embedding.Gemini.Model, cfg.Embedding.Gemini.APIKey, "")
	if err != nil {
		appLogger.Fatal(err.Error())
	}

	llmClient, err := llm.NewLLM(cfg.LLM.Provider, cfg.LLM.Gemini.Model, cfg.LLM.Gemini.APIKey, "", nil)
	if err != nil {
		appLogger.Fatal(err.Error())
	}

	// Initialize stores
	vecStore := store.NewMilvusStore(milvusClient, embedder, cfg.Databases.Milvus.Schema.CollectionName)
	graphStore := store.NewNeo4jStore(neo4jClient)

	// Initialize extractors
	factExtractor := extractor.NewLlmExtractor("python/extract_facts.py")
	graphExtractor := extractor.NewGraphExtractor(llmClient)

	// Initialize memory service
	memoryService := service.NewMemoryService(factExtractor, graphExtractor, vecStore, graphStore, llmClient, appLogger)

	// Initialize and start Kafka consumer
	kafkaConsumer := consumer.NewKafkaConsumer(kafkaClient, memoryService, appLogger)
	kafkaConsumer.Start(ctx)

	appLogger.Info("Memory service started")

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	appLogger.Info("Memory service stopped")
}
