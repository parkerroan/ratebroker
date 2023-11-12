package limiter

import (
	"time"
)

// RateLimitInfo contains information about the rate limit.
type RateLimitInfo struct {
	Remaining int           // Number of requests left for the time window
	Reset     time.Duration // Time remaining until the rate limit resets
	Limit     int           // Maximum number of requests allowed per time window
	Window    time.Duration // Time window for the rate limit
}

// Limiter is the interface that abstracts the limitations functionality.
type Limiter interface {
	// Accept logs a new request to the limiter.
	Accept(time.Time)
	// TryAccept checks if it's within thåe rate limits and logs a new request to the limiter.
	TryAccept(time.Time) bool
	// TryAccept checks if it's within the rate limits and logs a new request to the limiter.
	TryAcceptWithInfo(time.Time) (bool, RateLimitInfo)
	// LimitDetails returns the size and window of the limiter.
	LimitDetails() (int, time.Duration)
}
