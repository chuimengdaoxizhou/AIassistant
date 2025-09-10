package neo4j

import (
	"Jarvis_2.0/backend/go/internal/config"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var (
	instance *Neo4jClient
	once     sync.Once
	initErr  error
)

// Neo4jClient 包含了 Neo4j 驱动实例和从 YAML 加载的相关配置。
type Neo4jClient struct {
	Driver neo4j.DriverWithContext // Neo4j 驱动实例。
	Config *config.Neo4jConfig     // Neo4j 配置。
}

// GetClient 使用单例模式创建并返回一个新的 Neo4j 驱动实例。
func GetClient(ctx context.Context, cfg *config.Neo4jConfig) (*Neo4jClient, error) {
	once.Do(func() {
		// 使用用户名和密码创建认证 token。
		auth := neo4j.BasicAuth(cfg.Username, cfg.Password, "")

		// 创建驱动实例。
		driver, err := neo4j.NewDriverWithContext(cfg.Uri, auth)
		if err != nil {
			initErr = fmt.Errorf("无法创建 Neo4j 驱动: %w", err)
			return
		}

		// 验证与数据库的连接是否成功。
		if err := driver.VerifyConnectivity(ctx); err != nil {
			driver.Close(ctx) // 如果验证失败，需要关闭已创建的驱动以释放资源。
			initErr = fmt.Errorf("无法连接到 Neo4j 数据库: %w", err)
			return
		}

		log.Println("✅ 成功连接到 Neo4j!")
		instance = &Neo4jClient{Driver: driver, Config: cfg}
	})
	return instance, initErr
}

// Close 安全地关闭与 Neo4j 的连接。
func (c *Neo4jClient) Close(ctx context.Context) {
	if c.Driver != nil {
		if err := c.Driver.Close(ctx); err != nil {
			log.Printf("关闭 Neo4j 驱动失败: %v", err)
		}
	}
}

// HealthCheck 检查 Neo4j 连接的健康状况。
func (c *Neo4jClient) HealthCheck(ctx context.Context) error {
	return c.Driver.VerifyConnectivity(ctx)
}

// RunCypherQuery 执行一个 Cypher 查询并返回结果。
func (c *Neo4jClient) RunCypherQuery(ctx context.Context, query string, params map[string]interface{}) (neo4j.ResultWithContext, error) {
	session := c.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to run Cypher query: %w", err)
	}
	return result, nil
}

// ReadCypherQuery 执行一个 Cypher 读查询并返回结果。
func (c *Neo4jClient) ReadCypherQuery(ctx context.Context, query string, params map[string]interface{}) (neo4j.ResultWithContext, error) {
	session := c.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to run Cypher read query: %w", err)
	}
	return result, nil
}

// ExecuteWrite 在一个自动管理的写事务中执行 Cypher 查询。
func (c *Neo4jClient) ExecuteWrite(ctx context.Context, work func(tx neo4j.ManagedTransaction) (interface{}, error)) (interface{}, error) {
	session := c.Driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.Config.Database})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, work)
	if err != nil {
		return nil, fmt.Errorf("执行 Neo4j 写事务失败: %w", err)
	}

	return result, nil
}

// ExecuteRead 在一个自动管理的读事务中执行 Cypher 查询。
func (c *Neo4jClient) ExecuteRead(ctx context.Context, work func(tx neo4j.ManagedTransaction) (interface{}, error)) (interface{}, error) {
	session := c.Driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.Config.Database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, work)
	if err != nil {
		return nil, fmt.Errorf("执行 Neo4j 读事务失败: %w", err)
	}

	return result, nil
}

// CreateNode 创建一个带有指定标签和属性的新节点。
func (c *Neo4jClient) CreateNode(ctx context.Context, label string, properties map[string]interface{}) (int64, error) {
	result, err := c.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		cypherQuery := fmt.Sprintf("CREATE (n:%s $props) RETURN id(n)", label)

		result, err := tx.Run(ctx, cypherQuery, map[string]interface{}{"props": properties})
		if err != nil {
			return nil, err
		}

		record, err := result.Single(ctx)
		if err != nil {
			return nil, err
		}

		id, ok := record.Get("id(n)")
		if !ok {
			return nil, fmt.Errorf("无法从结果中获取 'id(n)'")
		}

		return id.(int64), nil
	})

	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// FindNodeByID 通过其内部 ID 查找节点，并返回其所有属性。
func (c *Neo4jClient) FindNodeByID(ctx context.Context, nodeID int64) (map[string]interface{}, error) {
	result, err := c.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		cypherQuery := "MATCH (n) WHERE id(n) = $id RETURN n"

		result, err := tx.Run(ctx, cypherQuery, map[string]interface{}{"id": nodeID})
		if err != nil {
			return nil, err
		}

		record, err := result.Single(ctx)
		if err != nil {
			return nil, err
		}

		nodeValue, ok := record.Get("n")
		if !ok {
			return nil, fmt.Errorf("无法从结果中获取 'n'")
		}

		node := nodeValue.(neo4j.Node)
		return node.Props, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(map[string]interface{}), nil
}
