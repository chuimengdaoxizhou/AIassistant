package ratelimiter

import (
	"sync"
	"time"
)

// TokenBucket implements the RateLimiter interface using the token bucket algorithm.
// It allows for bursts of requests up to the bucket's capacity.
type TokenBucket struct {
	rate          float64   // The rate at which tokens are generated (tokens per second).
	capacity      float64   // The maximum number of tokens in the bucket.
	tokens        float64   // The current number of tokens in the bucket.
	lastTokenTime time.Time // The last time tokens were added.
	mutex         sync.Mutex
}

// NewTokenBucket creates a new TokenBucket.
// rate: the number of tokens to generate per second.
// capacity: the maximum number of tokens (burst size).
func NewTokenBucket(rate float64, capacity int) *TokenBucket {
	return &TokenBucket{
		rate:          rate,
		capacity:      float64(capacity),
		tokens:        float64(capacity), // Start with a full bucket.
		lastTokenTime: time.Now(),
	}
}

// Allow checks if a request is allowed.
// It refills the bucket with new tokens based on the elapsed time
// and checks if at least one token is available.
func (tb *TokenBucket) Allow() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTokenTime)

	// Add new tokens based on the elapsed time.
	if elapsed > 0 {
		newTokens := elapsed.Seconds() * tb.rate
		tb.tokens = tb.tokens + newTokens
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastTokenTime = now
	}

	// Check if there is at least one token to consume.
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}

	return false
}
