package http

import (
	"Jarvis_2.0/backend/go/internal/config"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// helper function to create a mock config for testing
func newTestConfig() *config.AppConfig {
	return &config.AppConfig{
		Middleware: config.MiddlewareConfig{
			RateLimiter: config.RateLimiterConfig{
				Enabled:   true,
				Algorithm: "tokenBucket",
				TokenBucket: config.TokenBucketConfig{
					Rate:     10, // 10 tokens per second
					Capacity: 5,  // Bucket size of 5
				},
			},
			CircuitBreaker: config.CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 2, // Open after 2 consecutive failures
				SuccessThreshold: 2,
				Timeout:          "10s",
			},
		},
	}
}

func TestNewServer_WithAddress(t *testing.T) {
	cfg := newTestConfig()
	addr := ":9999"

	srv, err := NewServer(cfg, WithAddress(addr))
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	if srv.httpServer.Addr != addr {
		t.Errorf("Expected server address to be %s, but got %s", addr, srv.httpServer.Addr)
	}
}

func TestRateLimiterMiddleware(t *testing.T) {
	cfg := newTestConfig()
	// Use a very small capacity to make testing easier
	cfg.Middleware.RateLimiter.TokenBucket.Capacity = 2
	cfg.Middleware.RateLimiter.TokenBucket.Rate = 1

	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	srv.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	testServer := httptest.NewServer(srv.httpServer.Handler)
	defer testServer.Close()

	// First 2 requests should pass (equal to capacity)
	for i := 0; i < 2; i++ {
		resp, err := http.Get(testServer.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK on request %d, got %d", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// The 3rd request should be rate limited
	resp, err := http.Get(testServer.URL)
	if err != nil {
		t.Fatalf("Request 3 failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status TooManyRequests on request 3, got %d", resp.StatusCode)
	}
}

func TestCircuitBreakerMiddleware(t *testing.T) {
	cfg := newTestConfig()
	// Lower the failure threshold for the test
	cfg.Middleware.CircuitBreaker.FailureThreshold = 2

	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// This handler will always fail, to trip the breaker
	srv.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	})

	testServer := httptest.NewServer(srv.httpServer.Handler)
	defer testServer.Close()

	// First 2 requests should fail and trip the circuit
	for i := 0; i < 2; i++ {
		resp, err := http.Get(testServer.URL + "/fail")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status InternalServerError on request %d, got %d", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// The 3rd request should be blocked by the open circuit breaker
	resp, err := http.Get(testServer.URL + "/fail")
	if err != nil {
		t.Fatalf("Request 3 failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status ServiceUnavailable on request 3, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Circuit Breaker is open") {
		t.Errorf("Expected body to contain 'Circuit Breaker is open', got '%s'", string(body))
	}
}
