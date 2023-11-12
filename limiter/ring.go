package limiter

import (
	"container/ring"
	"sync"
	"time"
)

// RingLimiter is an implementation of the Limiter interface using a ring buffer.
// This is more performant than the HeapLimiter as it doesn't need to sort the requests by value and
// it uses a fixed size array.
type RingLimiter struct {
	ring   *ring.Ring
	size   int
	len    int
	window time.Duration
	mutex  sync.Mutex
}

// NewRingLimiterConstructorFunc returns a function that creates a new RingLimiter.
// This is used by default in the Broker.
func NewRingLimiterConstructorFunc() func(int, time.Duration) Limiter {
	return func(size int, window time.Duration) Limiter {
		return NewRingLimiter(size, window)
	}
}

// NewRingLimiterInstance creates a RingLimiter.
func NewRingLimiter(size int, window time.Duration) *RingLimiter {
	r := ring.New(size)
	return &RingLimiter{
		size:   size,
		ring:   r,
		window: window,
	}
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

// LimitDetails returns the size and window of the limiter.
func (rl *RingLimiter) LimitDetails() (int, time.Duration) {
	return rl.size, rl.window
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

	if rl.len < rl.size {
		rl.len++
	}
}

// TryAccept checks if it's within the rate limits, adds a new request to the ring buffer, and
// returns a boolean indicating whether the request was within the limits along with the rate limit info.
func (rl *RingLimiter) TryAcceptWithInfo(now time.Time) (bool, RateLimitInfo) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	info := rl.getRateLimitInfo(now) // Get the current rate limit info.

	// If we haven't reached the limit, we can accept the request.
	if info.Remaining > 0 {
		rl.accept(now) // Add the current request's timestamp to the ring.

		// Update the rate limit info to account for the newly accepted request.
		info.Remaining--
		// Note: If you want the Reset field to reflect the time until the oldest request
		// in the window expires, you'll need to calculate this based on the timestamps in the ring.
		// This example assumes a fixed window size.

		return true, info
	}

	// If the limit has been reached, the request isn't accepted.
	return false, info
}

func (rl *RingLimiter) calculateRemaining(oldestAllowedTime time.Time) int {
	invalidCount := 0

	// Start from the oldest request, which is the current position of the ring.
	for p := rl.ring; p.Value == nil || p.Value.(time.Time).Before(oldestAllowedTime); p = p.Next() {
		invalidCount++
		if p.Next() == rl.ring {
			// We've made a full circle, stop here.
			break
		}
	}

	// The number of remaining slots is the ring's capacity minus the number of invalid (old) entries.
	return invalidCount
}

func (rl *RingLimiter) getRateLimitInfo(now time.Time) RateLimitInfo {
	// Calculate the oldest point in time that requests should fall within.
	oldestAllowedTime := now.Add(-rl.window)

	info := RateLimitInfo{
		Limit: rl.size,
	}

	// Check the oldest request timestamp.
	if oldestRequest, ok := rl.ring.Value.(time.Time); ok && oldestRequest.After(oldestAllowedTime) {
		// All requests in the buffer are within the window.
		info.Remaining = 0
		// Calculate the time when the oldest request will no longer be considered for rate limiting.
		info.Reset = oldestRequest.Add(rl.window).Sub(now)
	} else {
		// The ring is empty; full number of requests are available.
		info.Remaining = rl.calculateRemaining(oldestAllowedTime)
		info.Reset = 0 // No wait needed before making a new request.
	}

	return info
}
