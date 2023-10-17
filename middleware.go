package ratebroker

import (
	"net/http"
)

// RateLimitHttpMiddleware is an HTTP middleware that can be used to rate limit requests.
func RateLimitHttpMiddleware(rb *RateBroker, keyGetter func(r *http.Request) string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userKey := keyGetter(r)
		// keyGetter has been removed as it was unused in your original snippet.
		// Supposing that TryAccept method checks IP-based rate limiting.
		if !rb.TryAccept(userKey) {
			// Too many requests from this IP
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		// Call the next handler/middleware in the chain.
		next.ServeHTTP(w, r)
	})
}
