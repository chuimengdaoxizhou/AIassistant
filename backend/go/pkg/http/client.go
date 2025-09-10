package http

import (
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/pkg/circuitbreaker"
	"fmt"
	"net/http"
	"time"
)

// Client is a custom HTTP client that wraps the standard http.Client
// and provides built-in support for circuit breaking.
type Client struct {
	httpClient *http.Client
	breaker    circuitbreaker.CircuitBreaker
}

// NewClient creates a new Client with a circuit breaker configured.
func NewClient(cfg config.CircuitBreakerConfig) (*Client, error) {
	if !cfg.Enabled {
		// If the circuit breaker is not enabled, return a client that uses the default http.Client
		return &Client{httpClient: http.DefaultClient, breaker: nil}, nil
	}

	breaker, err := createCircuitBreaker(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Example timeout
		},
		breaker: breaker,
	}, nil
}

// Do executes an HTTP request with circuit breaker protection.
// It considers status codes >= 500 as failures.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.breaker == nil {
		return c.httpClient.Do(req)
	}

	var resp *http.Response
	var err error

	// The breaker's Execute function returns its own error, which might be ErrCircuitOpen
	// or the error from the operation itself.
	_, breakerErr := c.breaker.Execute(func() (interface{}, error) {
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		// Treat server-side errors as failures for the circuit breaker
		if resp.StatusCode >= http.StatusInternalServerError {
			return nil, fmt.Errorf("server error: received status code %d", resp.StatusCode)
		}

		return resp, nil
	})

	if breakerErr != nil {
		// If the breaker is open, return that specific error.
		// Otherwise, the error is the actual error from the http call or the status code check.
		return nil, breakerErr
	}

	return resp, nil
}
