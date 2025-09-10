package grpc

import (
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/pkg/circuitbreaker"
	"Jarvis_2.0/backend/go/pkg/grpcinterceptor"
	"Jarvis_2.0/backend/go/pkg/ratelimiter"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"time"
)

// Server 是一个自定义的 gRPC 服务器，封装了标准的 grpc.Server 并提供了内置的中间件支持。
type Server struct {
	grpcServer *grpc.Server
	address    string
}

// ServerOption 定义了用于配置 Server 的函数。
type ServerOption func(*Server)

// WithAddress 设置服务器监听的地址。
func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

// NewServer 根据提供的 AppConfig 和选项创建并配置一个新的 Server 实例。
// 它会自动应用配置中启用的限流和熔断拦截器。
func NewServer(cfg *config.AppConfig, opts ...ServerOption) (*Server, error) {
	var interceptors []grpc.UnaryServerInterceptor

	// 如果启用了限流器，则添加限流拦截器。
	if cfg.Middleware.RateLimiter.Enabled {
		limiter, err := createRateLimiter(cfg.Middleware.RateLimiter)
		if err != nil {
			return nil, fmt.Errorf("failed to create rate limiter: %w", err)
		}
		log.Printf("Enabling gRPC Rate Limiter middleware with algorithm: %s", cfg.Middleware.RateLimiter.Algorithm)
		interceptors = append(interceptors, grpcinterceptor.RateLimitUnaryInterceptor(limiter))
	}

	// 如果启用了熔断器，则添加熔断拦截器。
	if cfg.Middleware.CircuitBreaker.Enabled {
		breaker, err := createCircuitBreaker(cfg.Middleware.CircuitBreaker)
		if err != nil {
			return nil, fmt.Errorf("failed to create circuit breaker: %w", err)
		}
		log.Println("Enabling gRPC Circuit Breaker middleware.")
		interceptors = append(interceptors, grpcinterceptor.CircuitBreakUnaryInterceptor(breaker))
	}

	// 将所有拦截器链接起来，并创建一个 gRPC 服务器实例。
	g := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...))

	srv := &Server{
		grpcServer: g,
	}

	// 应用所有传入的选项。
	for _, opt := range opts {
		opt(srv)
	}

	// 如果没有提供地址，则设置一个默认地址。
	if srv.address == "" {
		srv.address = ":9090"
	}

	return srv, nil
}

// RegisterService 暴露底层的 gRPC RegisterService 方法，用于注册服务实现。
func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.grpcServer.RegisterService(desc, impl)
}

// ListenAndServe 开始监听并提供 gRPC 服务。
func (s *Server) ListenAndServe() error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.address, err)
	}
	log.Printf("Starting gRPC server on %s", s.address)
	return s.grpcServer.Serve(lis)
}

// GracefulStop 优雅地停止 gRPC 服务器。
func (s *Server) GracefulStop() {
	s.grpcServer.GracefulStop()
}

// GetGRPCServer 返回底层的 *grpc.Server 实例。
func (s *Server) GetGRPCServer() *grpc.Server {
	return s.grpcServer
}

// createRateLimiter 和 createCircuitBreaker 函数可以从 http 包中复制过来，因为它们是通用的。
// （为简洁起见，这里省略了这两个函数的代码，假设它们已存在且与 http 包中的相同）
func createRateLimiter(cfg config.RateLimiterConfig) (ratelimiter.RateLimiter, error) {
	algorithm := cfg.Algorithm
	if algorithm == "" {
		algorithm = "tokenBucket"
	}
	switch algorithm {
	case "tokenBucket":
		conf := cfg.TokenBucket
		return ratelimiter.NewTokenBucket(conf.Rate, conf.Capacity), nil
	case "leakyBucket":
		conf := cfg.LeakyBucket
		return ratelimiter.NewLeakyBucket(conf.Rate, conf.Capacity), nil
	case "fixedWindow":
		conf := cfg.FixedWindow
		window, err := time.ParseDuration(conf.Window)
		if err != nil {
			return nil, fmt.Errorf("invalid fixedWindow duration: %w", err)
		}
		return ratelimiter.NewFixedWindowCounter(conf.Limit, window), nil
	case "slidingLog":
		conf := cfg.SlidingLog
		window, err := time.ParseDuration(conf.Window)
		if err != nil {
			return nil, fmt.Errorf("invalid slidingLog duration: %w", err)
		}
		return ratelimiter.NewSlidingWindowLog(conf.Limit, window), nil
	case "slidingCounter":
		conf := cfg.SlidingCounter
		window, err := time.ParseDuration(conf.Window)
		if err != nil {
			return nil, fmt.Errorf("invalid slidingCounter duration: %w", err)
		}
		return ratelimiter.NewSlidingWindowCounter(conf.Limit, window, conf.NumBuckets), nil
	default:
		return nil, fmt.Errorf("unknown rate limiter algorithm: %s", cfg.Algorithm)
	}
}

func createCircuitBreaker(cfg config.CircuitBreakerConfig) (circuitbreaker.CircuitBreaker, error) {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid circuit breaker timeout duration: %w", err)
	}
	return circuitbreaker.New(cfg.FailureThreshold, cfg.SuccessThreshold, timeout), nil
}
