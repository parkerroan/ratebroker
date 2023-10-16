package ring

import (
	"container/ring"
	"sync"
	"time"
)

// RingLimiter is an implementation of the Limiter interface using a ring buffer.
type RingLimiter struct {
	ring   *ring.Ring
	window time.Duration
	mutex  sync.Mutex
}

// NewRingLimiter creates a RingLimiter.
func NewRingLimiter(size int, window time.Duration) *RingLimiter {
	r := ring.New(size)
	return &RingLimiter{
		ring:   r,
		window: window,
	}
}

// Try checks if it's within the rate limits.
func (rl *RingLimiter) Try(now time.Time) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	return rl.try(now)
}

// Accept adds a new request to the ring buffer.
func (rl *RingLimiter) Accept(now time.Time) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.accept(now)
}

// TryAccept checks if it's within the rate limits and adds a new request to the ring buffer.
func (rl *RingLimiter) TryAccept(now time.Time) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if allowed := rl.try(now); allowed {
		rl.accept(now)
		return true
	}

	return false
}

// Try checks if it's within the rate limits.
func (rl *RingLimiter) try(now time.Time) bool {
	oldestAllowedTime := now.Add(-rl.window)

	if rl.ring.Value == nil || rl.ring.Value.(time.Time).Before(oldestAllowedTime) {
		return true
	}

	return false
}

// Accept adds a new request to the ring buffer.
func (rl *RingLimiter) accept(now time.Time) {
	rl.ring.Value = now
	rl.ring = rl.ring.Next()
}
