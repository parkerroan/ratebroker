package ratebroker_test

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/parkerroan/ratebroker"
	"github.com/parkerroan/ratebroker/limiter"
)

// ExampleHttpMiddleware shows how to use the middleware with a standard net/http handler or mux.
func ExampleHttpMiddleware() {
	// Initialize components of your application here
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Create instances of your broker and limiter
	redisBroker := ratebroker.NewRedisMessageBroker(rdb)

	// Create a rate broker w/ ring limiter
	rateBroker := ratebroker.NewRateBroker(
		ratebroker.WithBroker(redisBroker),
		ratebroker.WithMaxRequests(10),
		ratebroker.WithWindow(10*time.Second),
	)

	ctx := context.Background()
	rateBroker.Start(ctx)

	// This function generates a key (in this case, the client's IP address)
	// that the rate limiter uses to identify unique clients.
	keyGetter := func(r *http.Request) string {
		// You might want to improve this method to handle IP-forwarding, etc.
		return r.RemoteAddr
	}

	// Create a new router
	r := mux.NewRouter() // or http.NewServeMux()

	// Create a new rate limited HTTP handler using your middleware
	r.Use(ratebroker.HttpMiddleware(rateBroker, keyGetter))
}

// ExampleRateBroker_redisBroker shows how to create a rate broker with a Redis broker and ring limiter.
func ExampleRateBroker_redisBroker() {
	// Initialize components of your application here
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Create instances of your broker and limiter
	redisBroker := ratebroker.NewRedisMessageBroker(rdb)

	// Create a rate broker w/ ring limiter
	rateBroker := ratebroker.NewRateBroker(
		ratebroker.WithLimiterContructorFunc(limiter.NewRingLimiterConstructorFunc()), // or limiter.NewHeapLimiterConstructorFunc()
		ratebroker.WithMaxRequests(10),
		ratebroker.WithWindow(10*time.Second),
		ratebroker.WithBroker(redisBroker),
	)

	ctx := context.Background()
	// Start the broker in the background
	rateBroker.Start(ctx)

	for i := 0; i < 20; i++ {
		allowed, details := rateBroker.TryAccept(ctx, "userKey")
		log.Printf("Request %v allowed: %v details: %v", i, allowed, details)
	}
}

// ExampleRateBroker_localInstance shows how to create a rate broker without a memory broker.
func ExampleRateBroker_localInstance() {
	// Create a rate broker w/ ring limiter
	rateBroker := ratebroker.NewRateBroker(
		ratebroker.WithLimiterContructorFunc(limiter.NewRingLimiterConstructorFunc()), // or limiter.NewHeapLimiterConstructorFunc()
		ratebroker.WithMaxRequests(10),
		ratebroker.WithWindow(10*time.Second),
		//ratebroker.WithBroker(redisBroker), // Do not include a broker for local instance
	)

	ctx := context.Background()

	// Starting the broker in the background is not needed if you are not using a message broker.
	//rateBroker.Start(ctx)

	for i := 0; i < 20; i++ {
		allowed, details := rateBroker.TryAccept(ctx, "userKey")
		log.Printf("Request %v allowed: %v details: %v", i, allowed, details)
	}
}
