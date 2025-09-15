package agent

import (
	v1 "Jarvis_2.0/api/proto/v1"
)

// Registry 定义了 Agent 注册和发现的统一接口。
// AgentService 将依赖此接口，而不是具体的实现，以解决锁拷贝问题并实现依赖倒置。
type Registry interface {
	// GetAgentClient 获取到指定名称的子 Agent 的 gRPC 客户端。
	GetAgentClient(name string) (v1.AgentServiceClient, bool)

	// GetAgentMetadata 获取所有已注册的子 Agent 的元数据列表。
	GetAgentMetadata() []*v1.AgentMetadata

	// DiscoverAndCacheAgents 触发从服务发现（如etcd）中发现并缓存所有agent。
	DiscoverAndCacheAgents() error
}
