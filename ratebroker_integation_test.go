//go:build integration

package ratebroker_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/parkerroan/ratebroker"
	"github.com/parkerroan/ratebroker/limiter"
)

func TestRateLimiter_WithBroker(t *testing.T) {
	// Define the configuration for each test case.
	testCases := []struct {
		stream           string
		description      string
		limiterFunc      ratebroker.NewLimiterFunc
		window           time.Duration
		maxRequests      int
		numRequests      int
		preloadTopicNum  int
		expectedDenials  int
		sleepBetweenReqs time.Duration // Sleep duration between requests.
	}{
		{
			stream:           "test-stream1",
			description:      "Ring limiter should deny expected number of requests",
			limiterFunc:      limiter.NewRingLimiterConstructorFunc(),
			numRequests:      20,
			expectedDenials:  20, // Adjust based on your expected scenario
			preloadTopicNum:  20,
			sleepBetweenReqs: 20 * time.Millisecond,
			maxRequests:      5,
			window:           5 * time.Second,
		},
		{
			stream:           "test-stream2",
			description:      "Ring limiter should deny expected number of requests",
			limiterFunc:      limiter.NewRingLimiterConstructorFunc(),
			numRequests:      20,
			expectedDenials:  17, // Adjust based on your expected scenario
			preloadTopicNum:  2,
			sleepBetweenReqs: 20 * time.Millisecond,
			maxRequests:      5,
			window:           5 * time.Second,
		},
		{
			stream:           "test-stream3",
			description:      "Heap limiter should deny expected number of requests",
			limiterFunc:      limiter.NewHeapLimiterConstructorFunc(),
			numRequests:      20,
			expectedDenials:  20, // Adjust based on your expected scenario
			preloadTopicNum:  20,
			sleepBetweenReqs: 20 * time.Millisecond,
			maxRequests:      5,
			window:           5 * time.Second,
		},
		{
			stream:           "test-stream4",
			description:      "Heap limiter should deny expected number of requests",
			limiterFunc:      limiter.NewHeapLimiterConstructorFunc(),
			numRequests:      20,
			expectedDenials:  17, // Adjust based on your expected scenario
			preloadTopicNum:  2,
			sleepBetweenReqs: 20 * time.Millisecond,
			maxRequests:      5,
			window:           5 * time.Second,
		},
	}

	// Iterate through each test case.
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Context with timeout to avoid hanging tests indefinitely
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			//connect to redis
			rdb := redis.NewClient(&redis.Options{
				Addr: "localhost:6379",
			})

			// Create a new broker
			redisBroker := ratebroker.NewRedisMessageBroker(rdb, ratebroker.WithStream(tc.stream))

			rb := ratebroker.NewRateBroker(
				ratebroker.WithBroker(redisBroker),
				ratebroker.WithLimiterContructorFunc(tc.limiterFunc),
				ratebroker.WithWindow(tc.window),
				ratebroker.WithMaxRequests(tc.maxRequests),
			)

			rb.Start(ctx)

			for i := 0; i < tc.preloadTopicNum; i++ {
				//Publish a message
				msg := ratebroker.Message{
					Key:       "user1",
					Event:     ratebroker.RequestAccepted,
					Timestamp: time.Now(),
				}
				msg.BrokerID = fmt.Sprintf("test-ratebroker-%d", i)
				err := redisBroker.Publish(ctx, msg)
				if err != nil {
					t.Errorf("Failed to publish message: %v", err)
				}
			}

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
