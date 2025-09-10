package ratelimiter

// RateLimiter is the interface for rate limiting.
// It defines a single method, Allow, which returns true if a request is allowed,
// and false otherwise.
type RateLimiter interface {
	// Allow returns true if the request is allowed, otherwise returns false.
	Allow() bool
}
