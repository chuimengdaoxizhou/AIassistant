package etcd

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ServiceDiscovery defines the interface for service discovery.
type ServiceDiscovery struct {
	cli *clientv3.Client // etcd client
}

// NewServiceDiscovery creates a new ServiceDiscovery.
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

// Register registers a service with etcd.
func (s *ServiceDiscovery) Register(serviceName, addr string, ttl int64) (chan<- struct{}, error) {
	leaseResp, err := s.cli.Grant(context.Background(), ttl)
	if err != nil {
		return nil, err
	}

	_, err = s.cli.Put(context.Background(), "/"+serviceName+"/"+addr, addr, clientv3.WithLease(leaseResp.ID))
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
				return
			case _, ok := <-keepAliveCh:
				if !ok {
					// Lease expired or was revoked.
					s.revoke(serviceName, addr)
					return
				}
			}
		}
	}()

	return stop, nil
}

// revoke revokes a service from etcd.
func (s *ServiceDiscovery) revoke(serviceName, addr string) {
	// The lease will be automatically revoked by etcd, but we can also manually delete the key.
	s.cli.Delete(context.Background(), "/"+serviceName+"/"+addr)
}

// Discover discovers a service from etcd.
func (s *ServiceDiscovery) Discover(serviceName string) ([]string, error) {
	resp, err := s.cli.Get(context.Background(), "/"+serviceName, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var addrs []string
	for _, ev := range resp.Kvs {
		addrs = append(addrs, string(ev.Value))
	}

	return addrs, nil
}

// Close closes the etcd client.
func (s *ServiceDiscovery) Close() error {
	return s.cli.Close()
}
