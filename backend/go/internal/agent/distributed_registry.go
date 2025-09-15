package agent

import (
	v1 "Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/discovery/etcd"
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"sync"
	"time"
)

// DistributedRegistry 负责发现和管理分布式的子 Agent。
// 它实现了 Registry 接口。
var _ Registry = (*DistributedRegistry)(nil) // 编译时检查，确保实现了接口

type DistributedRegistry struct {
	sd      *etcd.ServiceDiscovery
	agents  map[string]*v1.AgentMetadata     // 修正：直接缓存从gRPC获取的protobuf元数据
	clients map[string]v1.AgentServiceClient // 缓存 gRPC 客户端连接
	mutex   sync.RWMutex
}

// NewDistributedRegistry 创建一个新的分布式注册表。
func NewDistributedRegistry(sd *etcd.ServiceDiscovery) *DistributedRegistry {
	return &DistributedRegistry{
		sd:      sd,
		agents:  make(map[string]*v1.AgentMetadata),
		clients: make(map[string]v1.AgentServiceClient),
	}
}

// DiscoverAndCacheAgents 从 etcd 发现所有 agent 服务并缓存它们的元数据和客户端连接。
func (r *DistributedRegistry) DiscoverAndCacheAgents() error {
	services, err := r.sd.DiscoverServices("/services/")
	if err != nil {
		return fmt.Errorf("failed to discover agents from etcd: %w", err)
	}

	for agentName, addr := range services {
		if _, ok := r.clients[agentName]; ok {
			continue
		}

		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("failed to connect to agent %s at %s: %v", agentName, addr, err)
			continue
		}

		client := v1.NewAgentServiceClient(conn)

		// 通过 gRPC 动态获取元数据
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		meta, err := client.GetMetadata(ctx, &empty.Empty{})
		if err != nil {
			log.Printf("failed to get metadata from agent %s: %v", agentName, err)
			cancel()
			conn.Close() // 获取元数据失败，关闭连接
			continue
		}
		cancel()

		r.mutex.Lock()
		r.agents[agentName] = meta
		r.clients[agentName] = client
		r.mutex.Unlock()

		log.Printf("Successfully discovered and cached agent: %s at %s", agentName, addr)
	}
	return nil
}

// GetAgentMetadata 返回所有缓存的 Agent 元数据。
func (r *DistributedRegistry) GetAgentMetadata() []*v1.AgentMetadata { // 修正：返回值类型
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var metadataList []*v1.AgentMetadata // 修正：切片类型
	for _, meta := range r.agents {
		metadataList = append(metadataList, meta)
	}
	return metadataList
}

// GetAgentClient 获取一个到指定 Agent 的 gRPC 客户端连接。
func (r *DistributedRegistry) GetAgentClient(name string) (v1.AgentServiceClient, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	client, found := r.clients[name]
	return client, found
}
