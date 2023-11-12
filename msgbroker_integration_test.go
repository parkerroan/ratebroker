package ratebroker_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/parkerroan/ratebroker"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
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

// TODO Fix
func TestRedisMessageBroker_Integration(t *testing.T) {
	// Set up a Redis client.
	// Note: For a real integration test, you might want to use a separate Redis instance (e.g., via Docker)
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_TEST_URL"), // use the correct address
	})

	// Ensure the connection is alive
	_, err := rdb.Ping(context.Background()).Result()
	assert.NoError(t, err)

	// Create a new Redis ratebroker
	broker := ratebroker.NewRedisMessageBroker(rdb,
		ratebroker.WithStream("redis-broker-integration-test-stream"),
	)

	// Context with timeout to avoid hanging tests indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Define a test message
	// remove nanoseconds from timestamp to avoid flaky tests
	now := time.Now().Truncate(time.Second)
	originalEvent := ratebroker.RateEvent{
		Timestamp: now,
		Key:       "test-key-1",
		BrokerID:  "test-ratebroker",
		Event:     ratebroker.RequestAccepted,
	}

	// Flag to check if the message was received
	messageReceived := make(chan struct{})

	// Test consuming the message
	go func() {
		broker.Start(ctx, func(msg ratebroker.RateEvent) {
			assert.Equal(t, originalEvent, msg, "Received message does not match the original")
			defer close(messageReceived) // signal that the message was received
			return
		})
	}()

	time.Sleep(1 * time.Second)

	// Test publishing the message
	err = broker.Publish(ctx, originalEvent)
	assert.NoError(t, err, "Failed to publish message")

	// Wait for the message to be received or timeout
	select {
	case <-messageReceived:
		// test succeeded
	case <-ctx.Done():
		t.Fatal("Test timed out before message was received")
	}
}
