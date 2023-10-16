package limiter

import "time"

// Limiter is the interface that abstracts the limitations functionality.
type Limiter interface {
	TryAccept(time.Time) bool
}
