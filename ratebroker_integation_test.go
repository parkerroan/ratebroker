package ratebroker_test

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/parkerroan/ratebroker"
	"github.com/parkerroan/ratebroker/limiter"
	"github.com/redis/go-redis/v9"
)

func init() {
	//load test.env file
	if _, err := os.Stat("test.env"); err == nil {
		// The file exists, now let's try to load it
		if err := godotenv.Load(); err != nil {
			// The file couldn't be loaded, log the error
			log.Fatalf("Error loading .env file: %s", err)
		}
	}
}

func BenchmarkIntegrationRateBroker_SingleUser_WithBroker(b *testing.B) {
	// Context with timeout to avoid hanging tests indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//connect to redis
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_TEST_URL"), // use the correct address
	})

	// Create a new broker
	redisBroker := ratebroker.NewRedisMessageBroker(rdb,
		ratebroker.WithStream("benchmark-test-stream"),
		ratebroker.WithMaxThreads(1000),
	)

	rb := ratebroker.NewRateBroker(
		ratebroker.WithBroker(redisBroker),
		ratebroker.WithLimiterContructorFunc(limiter.NewRingLimiterConstructorFunc()),
		ratebroker.WithWindow(2*time.Second),
		ratebroker.WithMaxRequests(5),
	)

	rb.Start(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.TryAccept(context.Background(), "user1")
	}
}

func BenchmarkIntegrationRateBroker_MultiUser_WithBroker(b *testing.B) {
	// Context with timeout to avoid hanging tests indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//connect to redis
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_TEST_URL"), // use the correct address
	})

	// Create a new broker
	redisBroker := ratebroker.NewRedisMessageBroker(rdb,
		ratebroker.WithStream("benchmark-test-stream"),
		ratebroker.WithCappedStream(100000),
		ratebroker.WithMaxThreads(100),
	)

	rb := ratebroker.NewRateBroker(
		ratebroker.WithBroker(redisBroker),
		ratebroker.WithLimiterContructorFunc(limiter.NewHeapLimiterConstructorFunc()),
		ratebroker.WithWindow(2*time.Second),
		ratebroker.WithMaxRequests(5),
	)

	rb.Start(ctx)
	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//pick a random int
		rb.TryAccept(context.Background(), fmt.Sprintf("user%v", rand.Intn(10000)))
	}
}

func TestIntegrationRateLimiter_WithBroker(t *testing.T) {
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
			description:      "Ring limiter should deny all requests",
			limiterFunc:      limiter.NewRingLimiterConstructorFunc(),
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
			description:      "Heap limiter should deny all requests",
			limiterFunc:      limiter.NewHeapLimiterConstructorFunc(),
			numRequests:      20,
			expectedDenials:  20, // Adjust based on your expected scenario
			preloadTopicNum:  20,
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
				Addr: os.Getenv("REDIS_TEST_URL"), // use the correct address
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

			//allow the broker to start
			time.Sleep(20 * time.Millisecond)

			for i := 0; i < tc.preloadTopicNum; i++ {
				//Publish a message
				msg := ratebroker.RateEvent{
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

			time.Sleep(10 * time.Millisecond)

			denied := 0
			for i := 0; i < tc.numRequests; i++ {
				if allowed := rb.TryAccept(context.Background(), "user1"); !allowed {
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
