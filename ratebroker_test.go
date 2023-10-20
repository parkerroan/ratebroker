//go:build unit

package ratebroker_test

import (
	"context"
	"testing"
	"time"

	"github.com/parkerroan/ratebroker"
	"github.com/parkerroan/ratebroker/limiter"
)

func TestRateLimiter(t *testing.T) {
	// Define the configuration for each test case.
	testCases := []struct {
		description      string
		limiterFunc      ratebroker.NewLimiterFunc
		window           time.Duration
		maxRequests      int
		numRequests      int
		expectedDenials  int
		sleepBetweenReqs time.Duration // Sleep duration between requests.
	}{
		{
			description:      "Ring limiter should deny expected number of requests",
			limiterFunc:      limiter.NewRingLimiterConstructorFunc(),
			numRequests:      20,
			expectedDenials:  15, // Adjust based on your expected scenario
			sleepBetweenReqs: 20 * time.Millisecond,
			maxRequests:      5,
			window:           2 * time.Second,
		},
		{
			description:      "Heap limiter should deny expected number of requests",
			limiterFunc:      limiter.NewHeapLimiterConstructorFunc(),
			numRequests:      20,
			expectedDenials:  15, // Adjust based on your expected scenario
			sleepBetweenReqs: 20 * time.Millisecond,
			maxRequests:      5,
			window:           2 * time.Second,
		},
	}

	// Iterate through each test case.
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			rb := ratebroker.NewRateBroker(
				ratebroker.WithLimiterContructorFunc(tc.limiterFunc),
				ratebroker.WithWindow(tc.window),
				ratebroker.WithMaxRequests(tc.maxRequests),
			)

			denied := 0
			for i := 0; i < tc.numRequests; i++ {
				if allowed, _ := rb.TryAccept(context.Background(), "user1"); !allowed {
					denied++
				}
				time.Sleep(tc.sleepBetweenReqs)
			}

			if denied != tc.expectedDenials {
				t.Errorf("Unexpected number of denied requests. Want: %d, got: %d", tc.expectedDenials, denied)
			}
		})
	}
}
