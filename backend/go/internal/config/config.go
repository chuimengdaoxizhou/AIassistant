package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// FieldConfig 定义了 Milvus 集合中字段的配置。
type FieldConfig struct {
	Name         string `yaml:"name"`                // 字段名称
	DataType     string `yaml:"dataType"`            // 字段数据类型 (例如: "Int64", "VarChar", "FloatVector")
	IsPrimaryKey bool   `yaml:"isPrimaryKey"`        // 是否为主键
	IsAutoID     bool   `yaml:"isAutoID"`            // 是否自动生成ID
	Dim          int    `yaml:"dim,omitempty"`       // 向量维度 (仅适用于向量类型)
	MaxLength    int    `yaml:"maxLength,omitempty"` // 最大长度 (仅适用于VarChar类型)
}

// IndexConfig 定义了 Milvus 集合中索引的配置。
type IndexConfig struct {
	FieldName  string                 `yaml:"fieldName"`  // 要创建索引的字段名称
	IndexType  string                 `yaml:"indexType"`  // 索引类型 (例如: "IVF_FLAT", "HNSW")
	MetricType string                 `yaml:"metricType"` // 相似度度量类型 (例如: "L2", "COSINE")
	Params     map[string]interface{} `yaml:"params"`     // 索引参数 (例如: {"nlist": 128})
}

// SchemaConfig 定义了 Milvus 集合的 Schema 配置。
type SchemaConfig struct {
	CollectionName string        `yaml:"collectionName"` // 集合名称
	Description    string        `yaml:"description"`    // 集合描述
	VectorField    string        `yaml:"vectorField"`    // 向量字段名称
	Fields         []FieldConfig `yaml:"fields"`         // 字段配置列表
	Index          IndexConfig   `yaml:"index"`          // 索引配置
}

// MilvusConfig 定义了 Milvus 数据库的连接和 Schema 配置。
type MilvusConfig struct {
	Address string       `yaml:"address"` // Milvus 服务地址
	Schema  SchemaConfig `yaml:"schema"`  // Milvus 集合 Schema 配置
}

// RedisConfig 定义了 Redis 数据库的连接配置。
type RedisConfig struct {
	Address  string `yaml:"address"`  // Redis 服务器地址 (例如: "localhost:6379")
	Password string `yaml:"password"` // Redis 密码
	DB       int    `yaml:"db"`       // Redis 数据库编号
}

// MySQLConfig 定义了 MySQL 数据库的连接配置。
type MySQLConfig struct {
	Address         string `yaml:"address"`         // MySQL 服务器地址
	Username        string `yaml:"username"`        // 用户名
	Password        string `yaml:"password"`        // 密码
	Database        string `yaml:"database"`        // 数据库名称
	MaxOpenConns    int    `yaml:"maxOpenConns"`    // 最大打开连接数
	MaxIdleConns    int    `yaml:"maxIdleConns"`    // 最大空闲连接数
	ConnMaxLifetime int    `yaml:"connMaxLifetime"` // 连接最大生命周期 (秒)
}

// MinIOConfig 定义了 MinIO 对象存储的连接配置。
type MinIOConfig struct {
	Endpoint  string `yaml:"endpoint"`  // MinIO 服务端点
	AccessKey string `yaml:"accessKey"` // 访问密钥
	SecretKey string `yaml:"secretKey"` // Secret 密钥
	Bucket    string `yaml:"bucket"`    // 默认存储桶名称
	Secure    bool   `yaml:"secure"`    // 是否使用HTTPS
}

// MongoConfig 定义了 MongoDB 数据库的连接配置。
type MongoConfig struct {
	Address  string `yaml:"address"`  // MongoDB 服务器地址
	Username string `yaml:"username"` // 用户名
	Password string `yaml:"password"` // 密码
	Database string `yaml:"database"` // 数据库名称
}

// Neo4jConfig 定义了 Neo4j 图数据库的连接配置。
type Neo4jConfig struct {
	Uri      string `yaml:"uri"`      // Neo4j 数据库URI (例如: "bolt://localhost:7687")
	Username string `yaml:"username"` // 用户名
	Password string `yaml:"password"` // 密码
	Database string `yaml:"database"` // 数据库名称
}

// MemgraphConfig 定义了 Memgraph 图数据库的连接配置。
type MemgraphConfig struct {
	Host     string `yaml:"host"`     // Memgraph 主机地址
	Port     int    `yaml:"port"`     // Memgraph 端口
	Username string `yaml:"username"` // 用户名
	Password string `yaml:"password"` // 密码
}

// EtcdConfig 定义了 Etcd 服务发现的连接配置。
type EtcdConfig struct {
	Endpoints []string `yaml:"endpoints"` // Etcd 节点地址列表
	Username  string   `yaml:"username"`  // 用户名
	Password  string   `yaml:"password"`  // 密码
}

// GoogleOAuthConfig 定义了 Google OAuth 的认证配置。
type GoogleOAuthConfig struct {
	ClientID     string `yaml:"clientID"`     // Google OAuth 客户端ID
	ClientSecret string `yaml:"clientSecret"` // Google OAuth 客户端Secret
	RedirectURL  string `yaml:"redirectURL"`  // 重定向URL
}

// AuthConfig 用于配置认证方法和相关设置。
type AuthConfig struct {
	Method     string            `yaml:"method"`     // 认证方法, "jwt" 或 "session"
	JwtSecret  string            `yaml:"jwtSecret"`  // JWT 密钥
	SessionKey string            `yaml:"sessionKey"` // session 密钥
	TokenTTL   int               `yaml:"tokenTTL"`   // JWT 令牌的有效期（秒）
	SessionTTL int               `yaml:"sessionTTL"` // session 的有效期（秒）
	Google     GoogleOAuthConfig `yaml:"google"`     // Google OAuth 配置
}

// KafkaConfig 定义了 Kafka 消息队列的连接配置。
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"` // Kafka Broker 地址列表
	Topics  []string `yaml:"topics"`  // Kafka 主题列表
}

// DatabaseConfigs 包含所有数据库的配置。
type DatabaseConfigs struct {
	Milvus   MilvusConfig   `yaml:"milvus"`   // Milvus 数据库配置
	Redis    RedisConfig    `yaml:"redis"`    // Redis 数据库配置
	MySQL    MySQLConfig    `yaml:"mysql"`    // MySQL 数据库配置
	MinIO    MinIOConfig    `yaml:"minio"`    // MinIO 对象存储配置
	MongoDB  MongoConfig    `yaml:"mongodb"`  // MongoDB 数据库配置
	Neo4j    Neo4jConfig    `yaml:"neo4j"`    // Neo4j 数据库配置
	Memgraph MemgraphConfig `yaml:"memgraph"` // Memgraph 数据库配置
	Etcd     EtcdConfig     `yaml:"etcd"`     // Etcd 服务发现配置
	Kafka    KafkaConfig    `yaml:"kafka"`    // Kafka 消息队列配置
}

// AppInfo 对应 'app' 部分，包含应用程序的基本信息。
type AppInfo struct {
	Name        string `yaml:"name"`        // 应用程序名称
	Version     string `yaml:"version"`     // 应用程序版本
	Environment string `yaml:"environment"` // 运行环境 (例如: "development", "production")
}

// LoggerConfig 定义了日志记录器的配置。
type LoggerConfig struct {
	Level string `yaml:"level"` // 日志级别 (例如: "info", "debug", "warn", "error")
}

// AppConfig 是整个 YAML 文件的根结构，包含了应用程序的所有配置。
type AppConfig struct {
	App        AppInfo         `yaml:"app"`       // 应用程序信息
	Auth       AuthConfig      `yaml:"auth"`      // 认证配置
	LLM        LLMConfig       `yaml:"llm"`       // LLM 配置部分
	Embedding  EmbeddingConfig `yaml:"embedding"` // Embedding 配置部分
	Logger     LoggerConfig    `yaml:"logger"`    // 日志记录器配置
	Databases  DatabaseConfigs `yaml:"databases"` // 数据库配置
	Middleware MiddlewareConfig `yaml:"middleware"` // 中间件配置
}

// LLMConfig 包含了不同LLM提供商的配置。
type LLMConfig struct {
	Provider string       `yaml:"provider"` // LLM提供商 (例如: "gemini", "openai")
	Gemini   GeminiConfig `yaml:"gemini"`   // Gemini 模型配置
}

// EmbeddingConfig 包含了不同Embedding提供商的配置。
type EmbeddingConfig struct {
	Provider string       `yaml:"provider"` // Embedding提供商 (例如: "gemini", "openai")
	Gemini   GeminiConfig `yaml:"gemini"`   // Gemini 模型配置
}

// GeminiConfig 包含了 Gemini 模型的配置。
type GeminiConfig struct {
	APIKey string `yaml:"apiKey"` // Gemini API 密钥
	Model  string `yaml:"model"`  // Gemini 模型名称
}

// MiddlewareConfig 包含所有中间件的配置。
type MiddlewareConfig struct {
	RateLimiter    RateLimiterConfig    `yaml:"rateLimiter"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuitBreaker"`
}

// RateLimiterConfig 定义了限流器的配置。
type RateLimiterConfig struct {
	Enabled        bool                 `yaml:"enabled"`
	Algorithm      string               `yaml:"algorithm"` // 支持: "fixedWindow", "slidingLog", "slidingCounter", "leakyBucket", "tokenBucket"
	FixedWindow    FixedWindowConfig    `yaml:"fixedWindow"`
	SlidingLog     SlidingLogConfig     `yaml:"slidingLog"`
	SlidingCounter SlidingCounterConfig `yaml:"slidingCounter"`
	LeakyBucket    LeakyBucketConfig    `yaml:"leakyBucket"`
	TokenBucket    TokenBucketConfig    `yaml:"tokenBucket"`
}

// FixedWindowConfig 定义了固定窗口计数器算法的配置。
type FixedWindowConfig struct {
	Limit  int    `yaml:"limit"`
	Window string `yaml:"window"` // 例如: "1m", "30s"
}

// SlidingLogConfig 定义了滑动窗口日志算法的配置。
type SlidingLogConfig struct {
	Limit  int    `yaml:"limit"`
	Window string `yaml:"window"`
}

// SlidingCounterConfig 定义了滑动窗口计数器算法的配置。
type SlidingCounterConfig struct {
	Limit      int    `yaml:"limit"`
	Window     string `yaml:"window"`
	NumBuckets int    `yaml:"numBuckets"`
}

// LeakyBucketConfig 定义了漏桶算法的配置。
type LeakyBucketConfig struct {
	Rate     float64 `yaml:"rate"` // 每秒速率
	Capacity int     `yaml:"capacity"`
}

// TokenBucketConfig 定义了令牌桶算法的配置。
type TokenBucketConfig struct {
	Rate     float64 `yaml:"rate"` // 每秒速率
	Capacity int     `yaml:"capacity"`
}

// CircuitBreakerConfig 定义了熔断器的配置。
type CircuitBreakerConfig struct {
	Enabled          bool   `yaml:"enabled"`
	FailureThreshold uint32 `yaml:"failureThreshold"`
	SuccessThreshold uint32 `yaml:"successThreshold"`
	Timeout          string `yaml:"timeout"` // 例如: "30s"
}

// LoadConfig 函数从指定路径加载并解析 YAML 配置文件。
//
// 参数:
//
//	path: YAML 配置文件的路径。
//
// 返回值:
//
//	*AppConfig: 解析后的应用程序配置结构体。
//	error: 如果文件读取或解析失败，则返回错误。
func LoadConfig(path string) (*AppConfig, error) {
	// 读取 YAML 文件内容。
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取 YAML 文件 '%s': %w", path, err)
	}
	var cfg AppConfig // 声明一个AppConfig变量用于存储解析后的配置。
	// 将 YAML 内容解析到 cfg 结构体中。
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		return nil, fmt.Errorf("解析 YAML 文件失败: %w", err)
	}
	return &cfg, nil // 返回解析后的配置和nil错误。
}
