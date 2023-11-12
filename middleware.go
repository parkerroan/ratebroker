package ratebroker

import (
	"fmt"
	"net/http"
)

// HTTPMiddleware creates a new middleware function for rate limiting.
// This function is compatible with both standard net/http and mux handlers.
func HTTPMiddleware(rb *RateBroker, keyGetter func(r *http.Request) string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userKey := keyGetter(r) // get the unique identifier for the requester
			ctx := r.Context()

			allowed, details := rb.TryAcceptWithInfo(ctx, userKey)
			if !allowed {
				// Apply rate limit headers or other response properties here
				w.Header().Add("RateLimit-Limit", fmt.Sprintf("%v", details.Limit))
				w.Header().Add("RateLimit-Remaining", fmt.Sprintf("%v", details.Remaining))
				w.Header().Add("RateLimit-Reset", fmt.Sprintf("%v", details.Reset.Seconds()))
				w.Header().Add("RateLimit-Policy", fmt.Sprintf("%v;w=%v", details.Limit, details.Window.Seconds()))
				w.WriteHeader(http.StatusTooManyRequests)
				// You might want to write a response message indicating the rate limit has been hit
				return
			}

			// Proceed to the next handler if not rate-limited
			next.ServeHTTP(w, r)
		})
	}
}
