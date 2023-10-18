package broker_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/parkerroan/ratebroker/broker"
	"github.com/stretchr/testify/assert"
)

func TestRedisBroker(t *testing.T) {
	// Set up a Redis client.
	// Note: For a real integration test, you might want to use a separate Redis instance (e.g., via Docker)
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // use the correct address
	})

	// Ensure the connection is alive
	_, err := rdb.Ping(context.Background()).Result()
	assert.NoError(t, err)

	// Create a new Redis broker
	redisBroker := broker.NewRedisBroker(rdb)

	// Context with timeout to avoid hanging tests indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Define a test message
	originalMsg := broker.Message{
		BrokerID: "test-broker",
		Event:    broker.RequestAccepted,
	}

	// Flag to check if the message was received
	messageReceived := make(chan struct{})

	// Test consuming the message
	go func() {
		err := redisBroker.Consume(ctx, func(msg broker.Message) {
			assert.Equal(t, originalMsg, msg, "Received message does not match the original")
			close(messageReceived) // signal that the message was received
		})
		assert.NoError(t, err, "Failed to consume message")
	}()

	time.Sleep(3 * time.Second)

	// Test publishing the message
	err = redisBroker.Publish(ctx, originalMsg)
	assert.NoError(t, err, "Failed to publish message")

	// Wait for the message to be received or timeout
	select {
	case <-messageReceived:
		// test succeeded
	case <-ctx.Done():
		t.Fatal("Test timed out before message was received")
	}
}
