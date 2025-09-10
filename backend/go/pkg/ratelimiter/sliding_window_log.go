package ratelimiter

import (
	"container/list"
	"sync"
	"time"
)

// SlidingWindowLog implements the RateLimiter interface using the sliding window log algorithm.
// It keeps a log of request timestamps in a sliding window.
type SlidingWindowLog struct {
	limit  int           // Maximum number of requests allowed in the window.
	window time.Duration // The duration of the time window.
	log    *list.List    // List to store request timestamps.
	mutex  sync.Mutex
}

// NewSlidingWindowLog creates a new SlidingWindowLog.
// limit: the maximum number of requests allowed in the window.
// window: the duration of the time window.
func NewSlidingWindowLog(limit int, window time.Duration) *SlidingWindowLog {
	return &SlidingWindowLog{
		limit:  limit,
		window: window,
		log:    list.New(),
	}
}

// Allow checks if a request is allowed.
// It removes old timestamps from the log and checks if the current log size is within the limit.
func (swl *SlidingWindowLog) Allow() bool {
	swl.mutex.Lock()
	defer swl.mutex.Unlock()

	now := time.Now()
	boundary := now.Add(-swl.window)

	// Remove timestamps that are outside the window.
	for e := swl.log.Front(); e != nil; {
		next := e.Next()
		if e.Value.(time.Time).Before(boundary) {
			swl.log.Remove(e)
		} else {
			// Since timestamps are ordered, we can stop when we find one within the window.
			break
		}
		e = next
	}

	// If the number of requests in the window is less than the limit, allow and log the new request.
	if swl.log.Len() < swl.limit {
		swl.log.PushBack(now)
		return true
	}

	return false
}
