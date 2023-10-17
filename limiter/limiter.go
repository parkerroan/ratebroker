package limiter

import (
	"time"
)

// Limiter is the interface that abstracts the limitations functionality.
type Limiter interface {
	Try(time.Time) bool
	Accept(time.Time)
	TryAccept(time.Time) bool
	LimitDetails() (int, time.Duration)
}
