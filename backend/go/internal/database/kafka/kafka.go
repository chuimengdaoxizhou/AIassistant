package kafka

import (
	"Jarvis_2.0/backend/go/internal/config"
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaClient 持有 Kafka writer 和 reader 的单例实例。
type KafkaClient struct {
	Writer *kafka.Writer
	Reader *kafka.Reader
	Conn   *kafka.Conn // 用于管理的连接
	Config *config.KafkaConfig
}

var (
	client  *KafkaClient
	once    sync.Once
	initErr error
)

// GetClient 使用单例模式初始化并返回一个 KafkaClient 实例。
// 首次调用时，它会连接到 Kafka 并根据配置自动创建所有必需的主题。
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

		// 1. 建立管理连接
		conn, err := kafka.Dial("tcp", cfg.Brokers[0])
		if err != nil {
			initErr = fmt.Errorf("kafka 初始化连接失败: %w", err)
			return
		}

		// 2. 获取已存在的主题
		partitions, err := conn.ReadPartitions()
		if err != nil {
			initErr = fmt.Errorf("无法读取 Kafka 分区信息: %w", err)
			conn.Close()
			return
		}
		existingTopics := make(map[string]struct{})
		for _, p := range partitions {
			existingTopics[p.Topic] = struct{}{}
		}

		// 3. 遍历并创建不存在的主题
		var topicsToCreate []kafka.TopicConfig
		for _, topicName := range cfg.Topics {
			if _, exists := existingTopics[topicName]; !exists {
				log.Printf("主题 '%s' 不存在，准备创建...", topicName)
				topicsToCreate = append(topicsToCreate, kafka.TopicConfig{
					Topic:             topicName,
					NumPartitions:     1, // 使用默认值
					ReplicationFactor: 1, // 使用默认值
				})
			}
		}

		if len(topicsToCreate) > 0 {
			err = conn.CreateTopics(topicsToCreate...)
			if err != nil {
				initErr = fmt.Errorf("自动创建 Kafka 主题失败: %w", err)
				conn.Close()
				return
			}
			log.Printf("成功创建 %d 个 Kafka 主题。", len(topicsToCreate))
		}

		// 4. 创建用于生产和消费的 Writer 和 Reader
		writer := &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
			BatchSize:    100,
		}

		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     cfg.Brokers,
			GroupID:     "default-consumer-group",
			MinBytes:    10e3, // 10KB
			MaxBytes:    10e6, // 10MB
			MaxAttempts: 10,
			Dialer: &kafka.Dialer{
				Timeout: 10 * time.Second,
			},
		})

		log.Println("✅ 成功初始化 Kafka 客户端!")
		client = &KafkaClient{Writer: writer, Reader: reader, Conn: conn, Config: cfg}
	})

	return client, initErr
}

// Close 安全地关闭单例的 Kafka 连接。
func (c *KafkaClient) Close() error {
	if c == nil {
		return nil
	}
	var errs []error
	if c.Writer != nil {
		if err := c.Writer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭 Kafka writer 失败: %w", err))
		}
	}
	if c.Reader != nil {
		if err := c.Reader.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭 Kafka reader 失败: %w", err))
		}
	}
	if c.Conn != nil {
		if err := c.Conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭 Kafka 管理连接失败: %w", err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("关闭 Kafka 客户端时发生多个错误: %v", errs)
	}
	return nil
}

// HealthCheck 检查 Kafka 连接的健康状况。
func (c *KafkaClient) HealthCheck(ctx context.Context) error {
	if c == nil || c.Conn == nil {
		return fmt.Errorf("kafka 客户端未初始化，无法进行健康检查")
	}
	_, err := c.Conn.Controller()
	return err
}

// GetControllerInfo 返回 Kafka 控制器的信息。
func (c *KafkaClient) GetControllerInfo() (string, error) {
	if c == nil || c.Conn == nil {
		return "", fmt.Errorf("kafka 客户端未初始化")
	}
	controller, err := c.Conn.Controller()
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)), nil
}