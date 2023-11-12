package limiter

import (
	"time"

	"golang.org/x/time/rate"
)

// TokenLimiter is an implementation of the Limiter interface using the rate package.
type TokenLimiter struct {
	limiter *rate.Limiter
	size    int
	window  time.Duration
}

// NewRateLimiter constructs a RateLimiter. The size is the number of tokens
// that the bucket starts with, and the window is how often tokens are replenished.
func NewTokenLimiter(size int, window time.Duration) *TokenLimiter {
	// The rate limiter's limit is the rate of token replenishment.
	limit := rate.Every(window / time.Duration(size))

	// The burst is the maximum size of the token bucket.
	burst := size

	return &TokenLimiter{
		limiter: rate.NewLimiter(limit, burst),
		size:    size,
		window:  window,
	}
}

// Accept reserves a spot for an event, blocking until it can proceed.
func (r *TokenLimiter) Accept(now time.Time) {
	// Reserve one token. If no token is available, this will block until one can be obtained.
	_ = r.limiter.ReserveN(now, 1)
}

// TryAccept attempts to reserve a spot for an event at time now.
func (r *TokenLimiter) TryAccept(now time.Time) bool {
	ret, _ := r.TryAcceptWithInfo(now)
	return ret
}

// TryAcceptWithInfo attempts to reserve a spot for an event at time now.
func (r *TokenLimiter) TryAcceptWithInfo(now time.Time) (bool, RateLimitInfo) {
	// Attempt to reserve one token. If no token is available, this will not block.
	reservation := r.limiter.ReserveN(now, 1)
	rateLimitInfo := RateLimitInfo{
		Remaining: int(r.limiter.Tokens()),
		Limit:     r.size,
		Reset:     reservation.Delay(),
	}

	if !reservation.OK() || reservation.Delay() > 0 {
		// Not allowed to proceed (the bucket is empty)
		return false, rateLimitInfo
	}

	return true, rateLimitInfo
}

// LimitDetails returns the size and window of the limiter.
func (r *TokenLimiter) LimitDetails() (int, time.Duration) {
	return r.size, r.window
}
