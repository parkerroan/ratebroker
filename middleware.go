package ratebroker

import (
	"fmt"
	"net/http"
)

// HttpMiddleware creates a new middleware function for rate limiting.
// This function is compatible with both standard net/http and mux handlers.
//
// Example:
//
//	// Create a new rate broker w/ ring limiter
//	r := mux.NewRouter() // or http.NewServeMux()
//	// This function generates a key (in this case, the client's IP address)
//	// that the rate limiter uses to identify unique clients.
//	keyGetter := func(r *http.Request) string {
//		// You might want to improve this method to handle IP-forwarding, etc.
//		return r.RemoteAddr
//	}
//
//	// Create a new rate limited HTTP handler using your middleware
//	r.Use(ratebroker.HttpMiddleware(rateBroker, keyGetter))
func HttpMiddleware(rb *RateBroker, keyGetter func(r *http.Request) string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userKey := keyGetter(r) // get the unique identifier for the requester
			ctx := r.Context()

			allowed, details := rb.TryAccept(ctx, userKey)
			if !allowed {
				// Apply rate limit headers or other response properties here
				w.Header().Add("X-Rate-Limit-Limit", fmt.Sprintf("%v", details.MaxRequests))
				w.Header().Add("X-Rate-Limit-Duration", fmt.Sprintf("%v", details.Window))
				w.WriteHeader(http.StatusTooManyRequests)
				// You might want to write a response message indicating the rate limit has been hit
				return
			}

			// Proceed to the next handler if not rate-limited
			next.ServeHTTP(w, r)
		})
	}
}
