package http

import (
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/pkg/circuitbreaker"
	"Jarvis_2.0/backend/go/pkg/httpmiddleware"
	"Jarvis_2.0/backend/go/pkg/ratelimiter"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Middleware defines a function to wrap an http.Handler.
type Middleware func(http.Handler) http.Handler

// Server is a custom HTTP server that wraps the standard http.Server
// and provides built-in support for middleware.
type Server struct {
	httpServer *http.Server
	mux        *http.ServeMux
}

// ServerOption defines a function for configuring a Server.
type ServerOption func(*Server)

// WithAddress sets the address for the server to listen on.
func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.httpServer.Addr = addr
	}
}

// NewServer creates and configures a new Server instance based on the provided AppConfig and options.
// It automatically applies rate limiting and circuit breaking middleware if enabled in the config.
func NewServer(cfg *config.AppConfig, opts ...ServerOption) (*Server, error) {
	mux := http.NewServeMux()
	var handler http.Handler = mux

	// Chain middleware
	var middlewares []Middleware

	// Add Rate Limiter middleware if enabled
	if cfg.Middleware.RateLimiter.Enabled {
		limiter, err := createRateLimiter(cfg.Middleware.RateLimiter)
		if err != nil {
			return nil, fmt.Errorf("failed to create rate limiter: %w", err)
		}
		log.Printf("Enabling Rate Limiter middleware with algorithm: %s", cfg.Middleware.RateLimiter.Algorithm)
		middlewares = append(middlewares, httpmiddleware.RateLimit(limiter))
	}

	// Add Circuit Breaker middleware if enabled
	if cfg.Middleware.CircuitBreaker.Enabled {
		breaker, err := createCircuitBreaker(cfg.Middleware.CircuitBreaker)
		if err != nil {
			return nil, fmt.Errorf("failed to create circuit breaker: %w", err)
		}
		log.Println("Enabling Circuit Breaker middleware.")
		middlewares = append(middlewares, httpmiddleware.CircuitBreak(breaker))
	}

	// Apply all middlewares in reverse order
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	srv := &Server{
		httpServer: &http.Server{
			Handler: handler,
		},
		mux: mux,
	}

	// Apply all the options
	for _, opt := range opts {
		opt(srv)
	}

	// Set a default address if none was provided
	if srv.httpServer.Addr == "" {
		srv.httpServer.Addr = ":8080"
	}

	return srv, nil
}

// Handle registers the handler for the given pattern.
func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// HandleFunc registers the handler function for the given pattern.
func (s *Server) HandleFunc(pattern string, handler http.HandlerFunc) {
	s.mux.HandleFunc(pattern, handler)
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	if s.httpServer.Addr == "" {
		return fmt.Errorf("server address is not set")
	}
	log.Printf("Starting server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// createRateLimiter initializes a rate limiter based on the configuration.
func createRateLimiter(cfg config.RateLimiterConfig) (ratelimiter.RateLimiter, error) {
	algorithm := cfg.Algorithm
	if algorithm == "" {
		algorithm = "tokenBucket" // Default as requested
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

// createCircuitBreaker initializes a circuit breaker based on the configuration.
func createCircuitBreaker(cfg config.CircuitBreakerConfig) (circuitbreaker.CircuitBreaker, error) {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid circuit breaker timeout duration: %w", err)
	}
	return circuitbreaker.New(cfg.FailureThreshold, cfg.SuccessThreshold, timeout), nil
}
