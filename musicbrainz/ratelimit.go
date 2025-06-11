package musicbrainz

import (
	"sync"
	"time"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
}

// NewRateLimiter creates a new rate limiter with the specified interval between requests
func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{
		interval: interval,
		last:     time.Time{},
	}
}

// Wait blocks until the next request can be made according to the rate limit
func (r *RateLimiter) Wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if !r.last.IsZero() {
		// Calculate how long to wait
		elapsed := now.Sub(r.last)
		if elapsed < r.interval {
			// Sleep for the remaining time
			time.Sleep(r.interval - elapsed)
		}
	}
	r.last = time.Now()
}
