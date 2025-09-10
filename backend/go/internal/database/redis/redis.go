package redis

import (
	"Jarvis_2.0/backend/go/internal/config"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/go-redis/redis/v8"
)

var (
	client  *redis.Client
	once    sync.Once
	initErr error
)

// GetClient 使用单例模式初始化并返回一个 Redis 客户端实例。
// 它确保到 Redis 的连接在整个应用生命周期中只被建立一次。
func GetClient(cfg *config.RedisConfig) (*redis.Client, error) {
	once.Do(func() {
		// 使用配置创建 Redis 客户端。
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.Address,
			Password: cfg.Password,
			DB:       cfg.DB,
		})

		// 使用 Ping 检查连接是否成功。
		ctx := context.Background()
		if err := rdb.Ping(ctx).Err(); err != nil {
			initErr = fmt.Errorf("无法连接到 Redis: %w", err)
			return
		}

		log.Println("✅ 成功连接到 Redis!")
		client = rdb
	})

	return client, initErr
}

// Close 安全地关闭单例的 Redis 连接。
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}

// HealthCheck 检查 Redis 连接的健康状况。
func HealthCheck(ctx context.Context) error {
	if client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}
	return client.Ping(ctx).Err()
}
