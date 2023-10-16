package ratebroker_test

import (
	"testing"
	"time"

	"github.com/parkerroan/ratebroker"
	"github.com/parkerroan/ratebroker/limiter"
)

func TestRateLimiter(t *testing.T) {
	// Define the configuration for each test case.
	testCases := []struct {
		description      string
		limiterFunc      limiter.NewLimiterFunc
		window           time.Duration
		rate             int
		numRequests      int
		expectedDenials  int
		sleepBetweenReqs time.Duration // Sleep duration between requests.
	}{
		{
			description:      "Ring limiter should deny expected number of requests",
			limiterFunc:      limiter.NewRingLimiter,
			numRequests:      20,
			expectedDenials:  15, // Adjust based on your expected scenario
			sleepBetweenReqs: 20 * time.Millisecond,
		},
		{
			description:      "Heap limiter should deny expected number of requests",
			limiterFunc:      limiter.NewHeapLimiter,
			numRequests:      20,
			expectedDenials:  15, // Adjust based on your expected scenario
			sleepBetweenReqs: 20 * time.Millisecond,
		},
	}

	// Iterate through each test case.
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			rb := ratebroker.NewRateBroker(nil, tc.limiterFunc)

			denied := 0
			for i := 0; i < tc.numRequests; i++ {
				if !rb.TryAccept("user1") {
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
