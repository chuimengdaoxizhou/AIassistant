package kafka

import (
	"Jarvis_2.0/backend/go/internal/config"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaClient 持有 Kafka writer 和 reader 的单例实例。
type KafkaClient struct {
	Writer *kafka.Writer
	Reader *kafka.Reader
	Config *config.KafkaConfig
}

var (
	client  *KafkaClient
	once    sync.Once
	initErr error
)

// GetClient 使用单例模式初始化并返回一个 KafkaClient 实例。
// 该实例包含一个 writer 和一个 reader。
func GetClient(cfg *config.KafkaConfig) (*KafkaClient, error) {
	once.Do(func() {
		if len(cfg.Brokers) == 0 {
			initErr = fmt.Errorf("未配置 Kafka brokers")
			return
		}
		if len(cfg.Topics) == 0 {
			initErr = fmt.Errorf("未配置 Kafka topics")
			return
		}

		// 检查与 broker 的基本连接性
		conn, err := kafka.DialContext(context.Background(), "tcp", cfg.Brokers[0])
		if err != nil {
			initErr = fmt.Errorf("kafka 初始化健康检查失败: %w", err)
			return
		}
		conn.Close()

		// 创建 Kafka 写入器。
		writer := &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        cfg.Topics[0], // 假设第一个主题用于默认写入器。
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
			BatchSize:    100,
		}

		// 创建 Kafka 读取器。
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     cfg.Brokers,
			Topic:       cfg.Topics[0], // 假设第一个主题用于默认读取器。
			GroupID:     "default-consumer-group",
			MinBytes:    10e3, // 10KB
			MaxBytes:    10e6, // 10MB
			MaxAttempts: 10,
			Dialer: &kafka.Dialer{
				Timeout: 10 * time.Second,
			},
		})

		log.Println("✅ 成功初始化 Kafka 客户端!")
		client = &KafkaClient{Writer: writer, Reader: reader, Config: cfg}
	})

	return client, initErr
}

// Close 安全地关闭单例的 Kafka 连接。
func Close() error {
	if client == nil {
		return nil
	}
	var err error
	if client.Writer != nil {
		if wErr := client.Writer.Close(); wErr != nil {
			err = fmt.Errorf("关闭 Kafka writer 失败: %w", wErr)
		}
	}
	if client.Reader != nil {
		if rErr := client.Reader.Close(); rErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; 关闭 Kafka reader 失败: %v", err, rErr)
			} else {
				err = fmt.Errorf("关闭 Kafka reader 失败: %w", rErr)
			}
		}
	}
	return err
}

// HealthCheck 检查 Kafka 连接的健康状况。
func HealthCheck(ctx context.Context) error {
	if client == nil || client.Config == nil || len(client.Config.Brokers) == 0 {
		return fmt.Errorf("Kafka 客户端未配置，无法进行健康检查")
	}

	conn, err := kafka.DialContext(ctx, "tcp", client.Config.Brokers[0])
	if err != nil {
		return fmt.Errorf("Kafka 健康检查失败: %w", err)
	}
	return conn.Close()
}
