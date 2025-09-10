package agent

import (
	"Jarvis_2.0/backend/go/internal/discovery/etcd"
	"sync"
)

// 全局唯一的 Registry 实例
var instance *Registry

// 用于保证初始化代码只执行一次的 sync.Once 对象
var once sync.Once

// 在初始化过程中可能发生的错误
var registryErr error

// Registry 定义了 agent 注册和发现的接口。
// 现在的它是一个单例，包含了 etcd 服务发现的客户端。
type Registry struct {
	sd     *etcd.ServiceDiscovery
	agents map[string][]string
	mutex  sync.RWMutex
}

// GetInstance 是获取 Registry 单例的唯一入口。
// 首次调用时，它会使用传入的 etcd endpoints 进行初始化。
// 后续的所有调用将直接返回已创建的实例，并忽略传入的 endpoints 参数。
func GetInstance(endpoints []string) (*Registry, error) {
	// sync.Once 的 Do 方法会确保里面的函数在程序运行期间只被执行一次。
	// 即使多个 goroutine 同时调用 GetInstance，初始化也只会发生一次。
	once.Do(func() {
		// --- 这是只会执行一次的初始化代码 ---
		sd, err := etcd.NewServiceDiscovery(endpoints)
		if err != nil {
			registryErr = err // 保存初始化错误
			return
		}

		// 创建实例并赋值给包级别的变量 instance
		instance = &Registry{
			sd:     sd,
			agents: make(map[string][]string),
		}
	})

	// 返回在 Do 方法中创建的实例和可能发生的错误
	return instance, registryErr
}

// RegisterAgent registers an agent with the registry.
func (r *Registry) RegisterAgent(agentName, agentAddress string, ttl int64) (chan<- struct{}, error) {
	return r.sd.Register(agentName, agentAddress, ttl)
}

// DiscoverAgents discovers agents from the registry.
func (r *Registry) DiscoverAgents(agentName string) ([]string, error) {
	// 注意：这里的缓存逻辑可能需要与服务发现的 watch 机制结合，才能实时更新。
	// 为了简化，我们暂时保留原有的逻辑，但重点是 Registry 本身已成为单例。
	r.mutex.RLock()
	agents, ok := r.agents[agentName]
	r.mutex.RUnlock() // 尽早释放读锁

	if ok && len(agents) > 0 {
		return agents, nil
	}

	// 如果缓存未命中，则从 etcd 发现
	discoveredAgents, err := r.sd.Discover(agentName)
	if err != nil {
		return nil, err
	}

	// 更新本地缓存
	r.mutex.Lock()
	r.agents[agentName] = discoveredAgents
	r.mutex.Unlock()

	return discoveredAgents, nil
}

// Close closes the registry.
func (r *Registry) Close() error {
	return r.sd.Close()
}
