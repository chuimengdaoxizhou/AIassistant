package agent

import "sync"

// LocalRegistry 在内存中存储和管理 Agent 实例。
type LocalRegistry struct {
	agents map[string]Agent
	mutex  sync.RWMutex
}

// NewLocalRegistry 创建一个新的本地注册表实例。
func NewLocalRegistry() *LocalRegistry {
	return &LocalRegistry{
		agents: make(map[string]Agent),
	}
}

// Register 将一个 Agent 实例添加到注册表。
func (r *LocalRegistry) Register(agent Agent) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.agents[agent.Metadata().Name] = agent
}

// GetAgent 根据名称检索一个 Agent。
func (r *LocalRegistry) GetAgent(name string) (Agent, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	agent, found := r.agents[name]
	return agent, found
}

// ListAgentsMetadata 返回所有已注册 Agent 的元数据列表。
// 这正是 MainAgent 需要调用的函数。
func (r *LocalRegistry) ListAgentsMetadata() []AgentMetadata {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var metadataList []AgentMetadata
	for _, agent := range r.agents {
		metadataList = append(metadataList, agent.Metadata())
	}
	return metadataList
}
