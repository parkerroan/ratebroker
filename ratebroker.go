package ratebroker

import (
	"fmt"
	"sync"
	"time"
)

// Limiter is the interface that abstracts the limitations functionality.
type Limiter interface {
	TryAccept(time.Time) bool
}

// RateLimiter is the main structure that will use a Limiter to enforce rate limits.
type RateLimiter struct {
	limiter     Limiter
	maxRequests int
	window      time.Duration
	mutex       sync.Mutex
}

// NewRateLimiter creates a RateLimiter with the provided Limiter.
func NewRateLimiter(limiter Limiter, maxRequests int, window time.Duration) *RateLimiter {
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

func main() {
	// Usage remains similar, but now we're injecting a RingLimiter into our RateLimiter.
	ringLimiter := NewRingLimiter(5, 10*time.Second)
	rateLimiter := NewRateLimiter(ringLimiter, 5, 10*time.Second)

	// ... rest of your code remains the same ...

	// Simulate incoming requests
	for i := 1; i <= 20; i++ {
		if rateLimiter.TryAccept() {
			fmt.Printf("Request %d accepted\n", i)
		} else {
			fmt.Printf("Request %d denied (rate limit exceeded)\n", i)
		}
		time.Sleep(1 * time.Second)
	}

	heapLimiter := NewHeapLimiter(5, 10*time.Second)
	rateLimiter = NewRateLimiter(heapLimiter, 5, 10*time.Second)

	// Simulate incoming requests
	for i := 1; i <= 20; i++ {
		if rateLimiter.TryAccept() {
			fmt.Printf("Request %d accepted\n", i)
		} else {
			fmt.Printf("Request %d denied (rate limit exceeded)\n", i)
		}
		time.Sleep(1 * time.Second)
	}
}
