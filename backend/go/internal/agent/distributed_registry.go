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
	"strconv"
	"strings"
	"sync"
)

// DistributedRegistry 负责发现和管理分布式的子 Agent。
type DistributedRegistry struct {
	sd         *etcd.ServiceDiscovery
	agents     map[string]AgentMetadata         // 缓存 Agent 的元数据
	clients    map[string]v1.AgentServiceClient // 缓存 gRPC 客户端连接
	clientLock sync.RWMutex
	metaLock   sync.RWMutex
}

// NewDistributedRegistry 创建一个新的分布式注册表。
func NewDistributedRegistry(sd *etcd.ServiceDiscovery) *DistributedRegistry {
	return &DistributedRegistry{
		sd:      sd,
		agents:  make(map[string]AgentMetadata),
		clients: make(map[string]v1.AgentServiceClient),
	}
}

// DiscoverAndCacheAgents 从 etcd 发现所有 agent 服务并缓存它们的元数据。
func (r *DistributedRegistry) DiscoverAndCacheAgents() error {
	// 这里的服务发现前缀可以做得更灵活
	services, err := r.sd.Discover("_agent")
	if err != nil {
		return fmt.Errorf("failed to discover agents from etcd: %w", err)
	}

	for serviceName, addresses := range services {
		if len(addresses) == 0 {
			continue
		}
		// 简单起见，我们只连接第一个地址
		addr := addresses[0]

		conn, err := grpc.Dial(strconv.Itoa(int(addr)), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("failed to connect to agent %s at %s: %v", serviceName, addr, err)
			continue
		}

		client := v1.NewAgentServiceClient(conn)
		meta, err := client.GetMetadata(context.Background(), &empty.Empty{})
		if err != nil {
			log.Printf("failed to get metadata from agent %s: %v", serviceName, err)
			conn.Close()
			continue
		}

		// 清理服务名称中的前缀，得到 agent 的真实名称
		agentName := strings.TrimSuffix(strconv.Itoa(serviceName), "_agent")

		// 缓存元数据和客户端
		r.metaLock.Lock()
		r.agents[agentName] = AgentMetadata{
			Name:              meta.GetName(),
			Capability:        meta.GetCapability(),
			InputDescription:  meta.GetInputDescription(),
			OutputDescription: meta.GetOutputDescription(),
		}
		r.metaLock.Unlock()

		r.clientLock.Lock()
		r.clients[agentName] = client
		r.clientLock.Unlock()

		log.Printf("Successfully discovered and cached agent: %s", agentName)
	}
	return nil
}

// GetAgentMetadata 返回所有缓存的 Agent 元数据。
func (r *DistributedRegistry) GetAgentMetadata() []AgentMetadata {
	r.metaLock.RLock()
	defer r.metaLock.RUnlock()
	var metadataList []AgentMetadata
	for _, meta := range r.agents {
		metadataList = append(metadataList, meta)
	}
	return metadataList
}

// GetAgentClient 获取一个到指定 Agent 的 gRPC 客户端连接。
func (r *DistributedRegistry) GetAgentClient(name string) (v1.AgentServiceClient, bool) {
	r.clientLock.RLock()
	defer r.clientLock.RUnlock()
	client, found := r.clients[name]
	return client, found
}
