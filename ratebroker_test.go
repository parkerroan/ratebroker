package ratebroker_test

import (
	"testing"
	"time"

	"github.com/parkerroan/ratebroker"
	"github.com/parkerroan/ratebroker/limiter"
	"github.com/parkerroan/ratebroker/limiter/heap"
	"github.com/parkerroan/ratebroker/limiter/ring"
)

func TestRateLimiter(t *testing.T) {
	// Define the configuration for each test case.
	testCases := []struct {
		description      string
		limiter          limiter.Limiter
		numRequests      int
		expectedDenials  int
		sleepBetweenReqs time.Duration // Sleep duration between requests.
	}{
		{
			description:      "Ring limiter should deny expected number of requests",
			limiter:          ring.NewRingLimiter(5, 2*time.Second),
			numRequests:      20,
			expectedDenials:  15, // Adjust based on your expected scenario
			sleepBetweenReqs: 20 * time.Millisecond,
		},
		{
			description:      "Heap limiter should deny expected number of requests",
			limiter:          heap.NewHeapLimiter(5, 2*time.Second),
			numRequests:      20,
			expectedDenials:  15, // Adjust based on your expected scenario
			sleepBetweenReqs: 20 * time.Millisecond,
		},
	}

	// Iterate through each test case.
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			rateLimiter := ratebroker.NewRateLimiter(tc.limiter, 5, 10*time.Second)

			denied := 0
			for i := 0; i < tc.numRequests; i++ {
				if !rateLimiter.TryAccept() {
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
