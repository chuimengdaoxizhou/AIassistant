package ratelimiter

import (
	"sync"
	"time"
)

// FixedWindowCounter implements the RateLimiter interface using a fixed window counter algorithm.
// It allows a certain number of requests in a fixed time window.
type FixedWindowCounter struct {
	limit       int           // Maximum number of requests allowed in the window.
	window      time.Duration // The duration of the time window.
	count       int           // Current number of requests in the window.
	windowStart time.Time     // The start time of the current window.
	mutex       sync.Mutex
}

// NewFixedWindowCounter creates a new FixedWindowCounter.
// limit: the maximum number of requests allowed in the window.
// window: the duration of the time window.
func NewFixedWindowCounter(limit int, window time.Duration) *FixedWindowCounter {
	return &FixedWindowCounter{
		limit:       limit,
		window:      window,
		windowStart: time.Now(),
	}
}

// Allow checks if a request is allowed.
// It resets the counter if the current time window has passed.
// It increments the counter if the request is within the limit.
func (fwc *FixedWindowCounter) Allow() bool {
	fwc.mutex.Lock()
	defer fwc.mutex.Unlock()

	now := time.Now()
	// If the window has passed, reset it.
	if now.After(fwc.windowStart.Add(fwc.window)) {
		fwc.windowStart = now
		fwc.count = 0
	}

	if fwc.count < fwc.limit {
		fwc.count++
		return true
	}

	return false
}
