package mysql

import (
	"Jarvis_2.0/backend/go/internal/config"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	dbInstance *gorm.DB
	once       sync.Once
	initErr    error
)

// GetDB 使用单例模式初始化并返回一个 GORM 数据库实例。
// 它确保数据库连接在整个应用生命周期中只被建立一次。
// 后续的调用将直接返回已存在的实例。
func GetDB(cfg *config.MySQLConfig) (*gorm.DB, error) {
	once.Do(func() {
		// 构建 DSN (Data Source Name) 字符串。
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.Username,
			cfg.Password,
			cfg.Address,
			cfg.Database,
		)

		// 使用 GORM 连接到 MySQL 数据库。
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			initErr = fmt.Errorf("无法连接到 MySQL: %w", err)
			return
		}

		// 获取底层 *sql.DB 实例，以便进行连接池配置。
		sqlDB, err := db.DB()
		if err != nil {
			initErr = fmt.Errorf("无法获取底层 SQL DB 实例: %w", err)
			return
		}

		// 配置连接池参数。
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

		log.Println("✅ 成功连接到 MySQL!")
		dbInstance = db
	})

	return dbInstance, initErr
}

// Close 安全地关闭单例的数据库连接。
func Close() error {
	if dbInstance != nil {
		sqlDB, err := dbInstance.DB()
		if err != nil {
			return fmt.Errorf("❌ 获取底层 SQL DB 实例失败: %w", err)
		}
		return sqlDB.Close()
	}
	return nil
}

// HealthCheck 检查数据库连接的健康状况。
func HealthCheck(ctx context.Context) error {
	if dbInstance == nil {
		return fmt.Errorf("数据库连接未初始化")
	}
	// 获取底层 *sql.DB 实例。
	sqlDB, err := dbInstance.DB()
	if err != nil {
		return fmt.Errorf("无法获取底层 SQL DB 实例进行健康检查: %w", err)
	}
	// Ping 数据库以检查连接性。
	return sqlDB.PingContext(ctx)
}
