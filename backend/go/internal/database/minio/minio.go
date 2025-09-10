package minio

import (
	"Jarvis_2.0/backend/go/internal/config"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	client  *minio.Client
	once    sync.Once
	initErr error
)

// GetClient 使用单例模式初始化并返回一个 MinIO 客户端实例。
// 它确保到 MinIO 的连接在整个应用生命周期中只被建立一次。
func GetClient(cfg *config.MinIOConfig) (*minio.Client, error) {
	once.Do(func() {
		// 使用配置中的端点、访问密钥和 Secret 密钥创建 MinIO 客户端。
		c, err := minio.New(cfg.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""), // 静态凭证。
			Secure: cfg.Secure,                                                // 是否使用 HTTPS。
		})
		if err != nil {
			initErr = fmt.Errorf("无法创建 MinIO 客户端: %w", err)
			return
		}

		// 初始化时执行简单的健康检查
		_, err = c.ListBuckets(context.Background())
		if err != nil {
			initErr = fmt.Errorf("MinIO 初始化健康检查失败: %w", err)
			return
		}

		log.Println("✅ 成功连接到 MinIO!")
		client = c
	})

	return client, initErr
}

// HealthCheck 检查 MinIO 连接的健康状况。
func HealthCheck(ctx context.Context) error {
	if client == nil {
		return fmt.Errorf("MinIO 客户端未初始化")
	}
	// 尝试列出存储桶以验证连接性和认证。
	_, err := client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("MinIO 健康检查失败: %w", err)
	}
	return nil
}

// Close 是一个占位符，因为 minio-go 客户端不需要显式关闭连接。
// 连接是按需创建和管理的。
func Close() {
	log.Println("ℹ️ MinIO 客户端资源已释放 (无需显式关闭)。")
}
