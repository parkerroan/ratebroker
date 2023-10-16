package limiter

import "time"

// Limiter is the interface that abstracts the limitations functionality.
type Limiter interface {
	Try(time.Time) bool
	Accept(time.Time)
}

// NewLimiterFunc creates a new limiter.
type NewLimiterFunc func(size int, window time.Duration) Limiter

func NewRingLimiter(size int, window time.Duration) Limiter {
	return NewRingLimiter(size, window)
}

func NewHeapLimiter(size int, window time.Duration) Limiter {
	return NewHeapLimiter(size, window)
}
