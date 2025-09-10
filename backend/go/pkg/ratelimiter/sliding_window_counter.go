package ratelimiter

import (
	"sync"
	"time"
)

// SlidingWindowCounter implements the RateLimiter interface using the sliding window counter algorithm.
// This approach is a compromise between the fixed window counter and the sliding window log,
// offering better memory efficiency than the log and better accuracy at window boundaries than the fixed counter.
type SlidingWindowCounter struct {
	limit        int           // Maximum number of requests allowed in the window.
	window       time.Duration // The total duration of the sliding window.
	numBuckets   int           // The number of buckets the window is divided into.
	bucketSize   time.Duration // The duration of a single bucket.
	buckets      []int         // Stores the count of requests for each bucket.
	currentBucket int           // Index of the current bucket.
	lastUpdateTime time.Time     // Timestamp of the last update.
	mutex        sync.Mutex
}

// NewSlidingWindowCounter creates a new SlidingWindowCounter.
// limit: the maximum number of requests allowed in the window.
// window: the duration of the time window.
// numBuckets: the number of buckets to divide the window into.
func NewSlidingWindowCounter(limit int, window time.Duration, numBuckets int) *SlidingWindowCounter {
	if numBuckets <= 0 {
		numBuckets = 10 // Default to 10 buckets if invalid value is provided.
	}
	return &SlidingWindowCounter{
		limit:        limit,
		window:       window,
		numBuckets:   numBuckets,
		bucketSize:   window / time.Duration(numBuckets),
		buckets:      make([]int, numBuckets),
		lastUpdateTime: time.Now(),
	}
}

// slideWindow slides the window forward in time, resetting buckets that are now outside the window.
func (swc *SlidingWindowCounter) slideWindow() {
	now := time.Now()
	elapsed := now.Sub(swc.lastUpdateTime)
	bucketsToSlide := int(elapsed / swc.bucketSize)

	if bucketsToSlide > 0 {
		// If we slide more than the total number of buckets, just reset all of them.
		if bucketsToSlide >= swc.numBuckets {
			for i := range swc.buckets {
				swc.buckets[i] = 0
			}
		} else {
			for i := 1; i <= bucketsToSlide; i++ {
				// Reset the bucket that is now out of the window
				nextBucket := (swc.currentBucket + i) % swc.numBuckets
				swc.buckets[nextBucket] = 0
			}
		}
		swc.currentBucket = (swc.currentBucket + bucketsToSlide) % swc.numBuckets
		swc.lastUpdateTime = now
	}
}

// Allow checks if a request is allowed.
func (swc *SlidingWindowCounter) Allow() bool {
	swc.mutex.Lock()
	defer swc.mutex.Unlock()

	swc.slideWindow()

	var totalCount int
	for _, count := range swc.buckets {
		totalCount += count
	}

	if totalCount < swc.limit {
		swc.buckets[swc.currentBucket]++
		return true
	}

	return false
}
