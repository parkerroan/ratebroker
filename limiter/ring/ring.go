package ring

import (
	"container/ring"
	"time"
)

// RingLimiter is an implementation of the Limiter interface using a ring buffer.
type RingLimiter struct {
	ring   *ring.Ring
	window time.Duration
}

// NewRingLimiter creates a RingLimiter.
func NewRingLimiter(size int, window time.Duration) *RingLimiter {
	r := ring.New(size)
	return &RingLimiter{
		ring:   r,
		window: window,
	}
}

// TryAccept records an attempt and checks if it's within the rate limits.
func (rl *RingLimiter) TryAccept(now time.Time) bool {
	oldestAllowedTime := now.Add(-rl.window)

	if rl.ring.Value == nil || rl.ring.Value.(time.Time).Before(oldestAllowedTime) {
		rl.ring.Value = now
		rl.ring = rl.ring.Next()
		return true
	}

	return false
}
