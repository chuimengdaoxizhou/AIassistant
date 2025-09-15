package milvus

import (
	"Jarvis_2.0/backend/go/internal/config"
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// InsertBatch inserts multiple records into the specified collection.
func (c *MilvusClient) InsertBatch(ctx context.Context, collectionName string, chunks []string, vectors [][]float32) error {
	if len(chunks) != len(vectors) {
		return fmt.Errorf("mismatch between number of chunks (%d) and vectors (%d)", len(chunks), len(vectors))
	}
	if len(chunks) == 0 {
		return nil // Nothing to insert
	}

	// Generate UUIDs for each chunk
	ids := make([]string, len(chunks))
	for i := range chunks {
		ids[i] = uuid.New().String()
	}

	// Assuming the schema has fields: "id" (VarChar, PK), "chunk" (VarChar), and "embedding" (FloatVector)
	idCol := entity.NewColumnVarChar("id", ids)
	chunkCol := entity.NewColumnVarChar("chunk", chunks)
	vectorCol := entity.NewColumnFloatVector("embedding", int64(len(vectors[0])), vectors)

	_, err := c.Client.Insert(ctx, collectionName, "" /* default partition */, idCol, chunkCol, vectorCol)
	if err != nil {
		return fmt.Errorf("failed to batch insert data into Milvus: %w", err)
	}

	log.Printf("✅ Successfully inserted %d records into collection '%s'.", len(chunks), collectionName)
	return nil
}

var (
	instance *MilvusClient
	once     sync.Once
	initErr  error
)

// MilvusClient 包含了 Milvus 客户端实例和相关配置。
type MilvusClient struct {
	Client client.Client        // Milvus 客户端实例。
	Config *config.MilvusConfig // Milvus 配置。
	// 【新增】用于控制后台自动刷新协程的取消函数。
	cancelAutoFlush context.CancelFunc
}

// GetClient 使用单例模式创建并返回一个 Milvus 客户端实例。
func GetClient(ctx context.Context, cfg *config.MilvusConfig) (*MilvusClient, error) {
	once.Do(func() {
		// 使用配置中的地址创建 Milvus 客户端。
		c, err := client.NewClient(ctx, client.Config{Address: cfg.Address})
		if err != nil {
			initErr = fmt.Errorf("无法连接到 Milvus: %w", err)
			return
		}
		log.Println("✅ 成功连接到 Milvus!")
		instance = &MilvusClient{Client: c, Config: cfg}
	})
	return instance, initErr
}

// Close 安全地关闭与 Milvus 的连接。
func (c *MilvusClient) Close() {
	if c.Client != nil {
		c.StopAutoFlush(context.Background()) // 使用一个独立的上下文来停止自动刷新。
		c.Client.Close()
		log.Println("ℹ️ 已安全关闭 Milvus 连接。")
	}
}

// HealthCheck 检查 Milvus 连接的健康状况。
func (c *MilvusClient) HealthCheck(ctx context.Context) error {
	if c.Client == nil {
		return fmt.Errorf("Milvus client is nil")
	}
	_, err := c.Client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("Milvus health check failed: %w", err)
	}
	return nil
}

// ErrNotFound 在搜索操作没有结果时返回。
var ErrNotFound = fmt.Errorf("not found")

// Search 在指定的分区中执行向量相似度搜索。
func (c *MilvusClient) Search(ctx context.Context, partitionName string, topK int, vector []float32) ([]client.SearchResult, error) {
	collName := c.Config.Schema.CollectionName

	if err := c.Client.LoadCollection(ctx, collName, false); err != nil {
		return nil, fmt.Errorf("加载集合 '%s' 失败: %w", collName, err)
	}
	log.Printf("✅ 集合 '%s' 已加载", collName)

	sp, _ := entity.NewIndexIvfFlatSearchParam(10)

	searchVectors := []entity.Vector{entity.FloatVector(vector)}

	log.Printf("⏳ 正在分区 '%s' 中搜索 (TopK=%d)...", partitionName, topK)
	results, err := c.Client.Search(
		ctx,
		collName,
		[]string{partitionName},
		"",
		[]string{"chunk"},
		searchVectors,
		c.Config.Schema.VectorField,
		entity.L2,
		topK,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("在分区 '%s' 中搜索失败: %w", partitionName, err)
	}

	log.Printf("✅ 搜索完成，找到 %d 个结果。", len(results))
	return results, nil
}

// Insert inserts a new memory record into the specified collection and partition.
func (c *MilvusClient) Insert(ctx context.Context, collectionName, partitionName, memoryID string, vector []float32) error {
	ids := []string{memoryID}
	vectors := [][]float32{vector}

	memoryIDCol := entity.NewColumnVarChar("memory_id", ids)
	vectorCol := entity.NewColumnFloatVector("embedding", c.Config.Schema.Fields[1].Dim, vectors)

	_, err := c.Client.Insert(ctx, collectionName, partitionName, memoryIDCol, vectorCol)
	if err != nil {
		return fmt.Errorf("failed to insert data into Milvus: %w", err)
	}

	return nil
}

// Delete deletes records from a Milvus collection based on a given ID.
func (c *MilvusClient) Delete(ctx context.Context, collectionName, partitionName, id string) error {
	expr := fmt.Sprintf("memory_id == \"%s\"", id)
	err := c.Client.Delete(ctx, collectionName, partitionName, expr)
	if err != nil {
		return fmt.Errorf("failed to delete data from Milvus: %w", err)
	}
	return nil
}

// CreatePartition 创建一个新的分区。
func (c *MilvusClient) CreatePartition(ctx context.Context, partitionName string) error {
	collName := c.Config.Schema.CollectionName
	err := c.Client.CreatePartition(ctx, collName, partitionName)
	if err != nil {
		return fmt.Errorf("为集合 '%s' 创建分区 '%s' 失败: %w", collName, partitionName, err)
	}
	log.Printf("✅ 成功创建分区: %s", partitionName)
	return nil
}

// HasPartition 检查指定的分区是否存在。
func (c *MilvusClient) HasPartition(ctx context.Context, partitionName string) (bool, error) {
	collName := c.Config.Schema.CollectionName
	partitions, err := c.Client.ShowPartitions(ctx, collName)
	if err != nil {
		return false, fmt.Errorf("无法获取集合 '%s' 的分区列表: %w", collName, err)
	}

	for _, p := range partitions {
		if p.Name == partitionName {
			return true, nil
		}
	}
	return false, nil
}

// DropPartition 删除一个分区。
func (c *MilvusClient) DropPartition(ctx context.Context, partitionName string) error {
	collName := c.Config.Schema.CollectionName
	err := c.Client.DropPartition(ctx, collName, partitionName)
	if err != nil {
		return fmt.Errorf("为集合 '%s' 删除分区 '%s' 失败: %w", collName, partitionName, err)
	}
	log.Printf("✅ 成功删除分区: %s", partitionName)
	return nil
}

// FlushCollection 手动触发一次刷新操作，将内存中的数据写入磁盘。
func (c *MilvusClient) FlushCollection(ctx context.Context) error {
	collName := c.Config.Schema.CollectionName
	log.Printf("⏳ 正在手动刷新集合 '%s'…", collName)
	if err := c.Client.Flush(ctx, collName, false); err != nil {
		return fmt.Errorf("刷新集合 '%s' 失败: %w", collName, err)
	}
	log.Printf("✅ 集合 '%s' 刷新成功！", collName)
	return nil
}

// StartAutoFlush 启动后台自动刷新任务。
func (c *MilvusClient) StartAutoFlush(interval time.Duration) {
	if c.cancelAutoFlush != nil {
		log.Println("⚠️ 自动刷新任务已在运行中，无需重复启动。")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelAutoFlush = cancel
	collName := c.Config.Schema.CollectionName

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		log.Printf("🚀 已启动后台自动刷新任务，每隔 %s 刷新一次集合 '%s'。", interval, collName)

		for {
			select {
			case <-ctx.Done():
				log.Println("ℹ️ 自动刷新任务已停止。")
				return
			case <-ticker.C:
				flushCtx, flushCancel := context.WithTimeout(context.Background(), 10*time.Second)
				if err := c.Client.Flush(flushCtx, collName, false); err != nil {
					log.Printf("❌ 自动刷新集合 '%s' 失败: %v", collName, err)
				}
				flushCancel()
			}
		}
	}()
}

// StopAutoFlush 停止后台自动刷新任务，并执行最后一次刷新以确保数据一致性。
func (c *MilvusClient) StopAutoFlush(ctx context.Context) {
	if c.cancelAutoFlush != nil {
		c.cancelAutoFlush()
		c.cancelAutoFlush = nil

		log.Println("⏳ 正在执行最后一次刷新以确保数据同步...")
		if err := c.FlushCollection(ctx); err != nil {
			log.Printf("❌ 停止自动刷新时，最终刷新失败: %v", err)
		}
	}
}

// GenerateFloatVector 是一个辅助函数，用于创建指定维度的随机向量。
func (c *MilvusClient) GenerateFloatVector(dim int) []float32 {
	vec := make([]float32, dim)
	for i := range vec {
		vec[i] = rand.Float32()
	}
	return vec
}

// EnsureCollection 确保 Milvus 集合存在并进行 Schema 迁移。
func (c *MilvusClient) EnsureCollection(ctx context.Context) error {
	collName := c.Config.Schema.CollectionName
	exists, err := c.Client.HasCollection(ctx, collName)
	if err != nil {
		return fmt.Errorf("检查集合是否存在时出错: %w", err)
	}
	if exists {
	} else {
		schemaFields := make([]*entity.Field, 0, len(c.Config.Schema.Fields))
		for _, fieldCfg := range c.Config.Schema.Fields {
			field := entity.NewField().WithName(fieldCfg.Name)

			if fieldCfg.IsPrimaryKey {
				field = field.WithIsPrimaryKey(true)
			}
			if fieldCfg.IsAutoID {
				field = field.WithIsAutoID(true)
			}

			switch fieldCfg.DataType {
			case "Int64":
				field = field.WithDataType(entity.FieldTypeInt64)
			case "VarChar":
				field = field.WithDataType(entity.FieldTypeVarChar).WithMaxLength(int64(fieldCfg.MaxLength))
			case "FloatVector":
				field = field.WithDataType(entity.FieldTypeFloatVector).WithDim(int64(fieldCfg.Dim))
			case "BinaryVector":
				field = field.WithDataType(entity.FieldTypeBinaryVector).WithDim(int64(fieldCfg.Dim))
			case "Float":
				field = field.WithDataType(entity.FieldTypeFloat)
			case "Double":
				field = field.WithDataType(entity.FieldTypeDouble)
			case "Bool":
				field = field.WithDataType(entity.FieldTypeBool)
			default:
				return fmt.Errorf("不支持的数据类型: %s", fieldCfg.DataType)
			}

			schemaFields = append(schemaFields, field)
		}

		schema := entity.NewSchema().
			WithName(collName).
			WithDescription(c.Config.Schema.Description)

		for _, field := range schemaFields {
			schema = schema.WithField(field)
		}

		if err := c.Client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
			return fmt.Errorf("创建集合失败: %w", err)
		}
		idx, err := c.buildIndexFromConfig()
		if err != nil {
			return err
		}
		if err := c.Client.CreateIndex(ctx, collName, c.Config.Schema.Index.FieldName, idx, false); err != nil {
			return fmt.Errorf("为字段 '%s' 创建索引失败: %w", c.Config.Schema.Index.FieldName, err)
		}
	}

	err = c.Client.LoadCollection(ctx, collName, false)
	if err != nil {
		return fmt.Errorf("加载 Milvus 集合 '%s' 失败: %w", collName, err)
	}

	return nil
}

// buildIndexFromConfig 是一个辅助函数，用于从配置构建索引实体。
func (c *MilvusClient) buildIndexFromConfig() (entity.Index, error) {
	indexCfg := c.Config.Schema.Index
	metricType := entity.MetricType(indexCfg.MetricType)

	switch indexCfg.IndexType {
	case "IVF_FLAT":
		nlist, ok := indexCfg.Params["nlist"].(int)
		if !ok {
			nlist = 128
		}
		return entity.NewIndexIvfFlat(metricType, nlist)
	case "HNSW":
		M, ok := indexCfg.Params["M"].(int)
		if !ok {
			M = 8
		}
		efConstruction, ok := indexCfg.Params["efConstruction"].(int)
		if !ok {
			efConstruction = 96
		}
		return entity.NewIndexHNSW(metricType, M, efConstruction)
	case "IVF_SQ8":
		nlist, ok := indexCfg.Params["nlist"].(int)
		if !ok {
			nlist = 128
		}
		return entity.NewIndexIvfSQ8(metricType, nlist)
	case "IVF_PQ":
		nlist, ok := indexCfg.Params["nlist"].(int)
		if !ok {
			nlist = 128
		}
		m, ok := indexCfg.Params["m"].(int)
		if !ok {
			m = 16
		}
		nbits, ok := indexCfg.Params["nbits"].(int)
		if !ok {
			nbits = 8
		}
		return entity.NewIndexIvfPQ(metricType, nlist, m, nbits)
	case "AUTOINDEX":
		return entity.NewIndexAUTOINDEX(metricType)
	default:
		return nil, fmt.Errorf("不支持的索引类型: %s", indexCfg.IndexType)
	}
}
