package httpmiddleware

import (
	"Jarvis_2.0/backend/go/pkg/circuitbreaker"
	"Jarvis_2.0/backend/go/pkg/ratelimiter"
	"fmt"
	"net/http"
)

// RateLimit is a middleware that applies rate limiting to an HTTP handler.
func RateLimit(limiter ratelimiter.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter is a wrapper for http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// CircuitBreak is a middleware that applies the circuit breaker pattern to an HTTP handler.
// It considers HTTP status codes >= 500 as failures.
func CircuitBreak(breaker circuitbreaker.CircuitBreaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap the response writer to capture the status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			_, err := breaker.Execute(func() (interface{}, error) {
				next.ServeHTTP(rw, r)

				// If the status code is a server error, report it as a failure to the circuit breaker.
				if rw.statusCode >= http.StatusInternalServerError {
					return nil, fmt.Errorf("server error: status code %d", rw.statusCode)
				}

				// Otherwise, it's a success.
				return nil, nil
			})

			if err != nil {
				if err == circuitbreaker.ErrCircuitOpen {
					// When the circuit is open, prevent the request and return Service Unavailable.
					http.Error(w, "Service Unavailable: Circuit Breaker is open", http.StatusServiceUnavailable)
					return
				}
				// Note: The original error (e.g., "server error: status code 500") is already written
				// to the response by the handler `next`. We just return here.
			}
		})
	}
}
