package etcd

import (
	"context"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ServiceDiscovery 定义了服务发现的接口。
type ServiceDiscovery struct {
	cli *clientv3.Client // etcd client
}

// NewServiceDiscovery 创建一个新的 ServiceDiscovery。
func NewServiceDiscovery(endpoints []string) (*ServiceDiscovery, error) {

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &ServiceDiscovery{cli: cli}, nil
}

// Register 注册一个服务到 etcd。
func (s *ServiceDiscovery) Register(serviceName, addr string, ttl int64) (chan<- struct{}, error) {
	leaseResp, err := s.cli.Grant(context.Background(), ttl)
	if err != nil {
		return nil, err
	}

	// 使用 /services/ 作为公共前缀，方便统一发现
	key := "/services/" + serviceName + "/" + addr
	_, err = s.cli.Put(context.Background(), key, addr, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		return nil, err
	}

	keepAliveCh, err := s.cli.KeepAlive(context.Background(), leaseResp.ID)
	if err != nil {
		return nil, err
	}

	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				// 停止续约并撤销租约
				s.revoke(leaseResp.ID)
				return
			case _, ok := <-keepAliveCh:
				if !ok {
					// 租约过期或被撤销
					return
				}
			}
		}
	}()

	return stop, nil
}

// revoke 撤销一个租约。
func (s *ServiceDiscovery) revoke(leaseID clientv3.LeaseID) {
	// 撤销租约将自动删除所有关联的键
	s.cli.Revoke(context.Background(), leaseID)
}

// Discover 发现一个特定名称的服务的所有实例地址。
func (s *ServiceDiscovery) Discover(serviceName string) ([]string, error) {
	key := "/services/" + serviceName + "/"
	resp, err := s.cli.Get(context.Background(), key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var addrs []string
	for _, ev := range resp.Kvs {
		addrs = append(addrs, string(ev.Value))
	}

	return addrs, nil
}

// DiscoverServices 发现给定前缀下的所有服务。
// 返回一个 map，键是服务名，值是该服务的第一个实例地址。
func (s *ServiceDiscovery) DiscoverServices(prefix string) (map[string]string, error) {
	resp, err := s.cli.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	services := make(map[string]string)
	for _, ev := range resp.Kvs {
		key := string(ev.Key)
		trimmedKey := strings.TrimPrefix(key, prefix)
		parts := strings.SplitN(trimmedKey, "/", 2)
		if len(parts) > 0 {
			serviceName := parts[0]
			if _, exists := services[serviceName]; !exists {
				services[serviceName] = string(ev.Value)
			}
		}
	}

	return services, nil
}

// Close 关闭 etcd 客户端。
func (s *ServiceDiscovery) Close() error {
	return s.cli.Close()
}