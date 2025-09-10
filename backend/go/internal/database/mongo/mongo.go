package mongo

import (
	"Jarvis_2.0/backend/go/internal/config"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client  *mongo.Client
	once    sync.Once
	initErr error
)

// GetClient 使用单例模式初始化并返回一个 MongoDB 客户端实例。
// 它确保到 MongoDB 的连接在整个应用生命周期中只被建立一次。
func GetClient(cfg *config.MongoConfig) (*mongo.Client, error) {
	once.Do(func() {
		// 应用连接URI。
		clientOptions := options.Client().ApplyURI(cfg.Address)
		// 如果配置了用户名和密码，则设置认证信息。
		if cfg.Username != "" && cfg.Password != "" {
			clientOptions.SetAuth(options.Credential{
				Username: cfg.Username,
				Password: cfg.Password,
			})
		}

		// 创建一个带有超时功能的上下文。
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel() // 确保在函数退出时取消上下文。

		// 连接到 MongoDB。
		c, err := mongo.Connect(ctx, clientOptions)
		if err != nil {
			initErr = fmt.Errorf("无法连接到 MongoDB: %w", err)
			return
		}

		// 检查连接是否成功（Ping 数据库）。
		if err = c.Ping(ctx, nil); err != nil {
			initErr = fmt.Errorf("无法 Ping MongoDB: %w", err)
			return
		}

		log.Println("✅ 成功连接到 MongoDB!")
		client = c
	})

	return client, initErr
}

// Close 安全地断开单例的 MongoDB 客户端连接。
func Close(ctx context.Context) error {
	if client != nil {
		return client.Disconnect(ctx)
	}
	return nil
}

// HealthCheck 检查 MongoDB 连接的健康状况。
func HealthCheck(ctx context.Context) error {
	if client == nil {
		return fmt.Errorf("MongoDB 客户端未初始化")
	}
	return client.Ping(ctx, nil)
}
