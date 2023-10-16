package ratebroker

import (
	"sync"
	"time"

	"github.com/parkerroan/ratebroker/limiter"
)

// RateLimiter is the main structure that will use a Limiter to enforce rate limits.
type RateLimiter struct {
	limiter     limiter.Limiter
	maxRequests int
	window      time.Duration
	mutex       sync.Mutex
}

// NewRateLimiter creates a RateLimiter with the provided Limiter.
func NewRateLimiter(limiter limiter.Limiter, maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limiter:     limiter,
		maxRequests: maxRequests,
		window:      window,
	}
}

// TryAccept is a method on RateLimiter that checks a new request against the current rate limit.
func (rl *RateLimiter) TryAccept() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	return rl.limiter.TryAccept(now)
}
