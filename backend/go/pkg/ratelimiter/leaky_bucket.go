package ratelimiter

import (
	"sync"
	"time"
)

// LeakyBucket implements the RateLimiter interface using the leaky bucket algorithm.
// It ensures a steady outflow of requests, smoothing out bursts.
type LeakyBucket struct {
	rate         float64   // The rate at which the bucket leaks (requests per second).
	capacity     float64   // The maximum capacity of the bucket.
	waterLevel   float64   // The current "water level" (number of requests) in the bucket.
	lastLeakTime time.Time // The last time the bucket was leaked.
	mutex        sync.Mutex
}

// NewLeakyBucket creates a new LeakyBucket.
// rate: the number of requests to process per second.
// capacity: the maximum burst size (bucket capacity).
func NewLeakyBucket(rate float64, capacity int) *LeakyBucket {
	return &LeakyBucket{
		rate:         rate,
		capacity:     float64(capacity),
		lastLeakTime: time.Now(),
	}
}

// Allow checks if a request is allowed.
// It calculates how much the bucket has "leaked" since the last request
// and determines if there is enough capacity for the new request.
func (lb *LeakyBucket) Allow() bool {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(lb.lastLeakTime)

	// Calculate how much water has leaked since the last check.
	leakedWater := elapsed.Seconds() * lb.rate
	if leakedWater > 0 {
		lb.waterLevel = lb.waterLevel - leakedWater
		if lb.waterLevel < 0 {
			lb.waterLevel = 0
		}
		lb.lastLeakTime = now
	}

	// Check if there is capacity for one more drop of water.
	if lb.waterLevel < lb.capacity {
		lb.waterLevel++
		return true
	}

	return false
}
