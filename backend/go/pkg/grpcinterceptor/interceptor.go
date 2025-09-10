package grpcinterceptor

import (
	"Jarvis_2.0/backend/go/pkg/circuitbreaker"
	"Jarvis_2.0/backend/go/pkg/ratelimiter"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RateLimitUnaryInterceptor 返回一个 gRPC 一元拦截器，用于限流。
func RateLimitUnaryInterceptor(limiter ratelimiter.RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !limiter.Allow() {
			// 当请求被限流时，返回 gRPC 标准的 ResourceExhausted 错误码。
			return nil, status.Errorf(codes.ResourceExhausted, "request rejected due to rate limiting")
		}
		return handler(ctx, req)
	}
}

// CircuitBreakUnaryInterceptor 返回一个 gRPC 一元拦截器，用于熔断。
func CircuitBreakUnaryInterceptor(breaker circuitbreaker.CircuitBreaker) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 将 gRPC handler 的调用包装在熔断器的 Execute 方法中。
		resp, err := breaker.Execute(func() (interface{}, error) {
			return handler(ctx, req)
		})

		if err != nil {
			// 如果熔断器已打开，返回 gRPC 标准的 Unavailable 错误码。
			if err == circuitbreaker.ErrCircuitOpen {
				return nil, status.Errorf(codes.Unavailable, "service unavailable: circuit breaker is open")
			}
			// 否则，返回原始错误。
			return nil, err
		}

		return resp, nil
	}
}
