package limiter

import (
	"time"
)

// Limiter is the interface that abstracts the limitations functionality.
type Limiter interface {
	// Try checks if it's within the rate limits.
	Try(time.Time) bool
	// Accept logs a new request to the limiter.
	Accept(time.Time)
	// TryAccept checks if it's within the rate limits and logs a new request to the limiter.
	TryAccept(time.Time) bool
	// LimitDetails returns the size and window of the limiter.
	LimitDetails() (int, time.Duration)
}
