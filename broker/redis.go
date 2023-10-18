package broker

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis/v8"
)

// Option is a function that can be passed into NewRateBroker to configure the RateBroker.
type Option func(*RedisBroker)

// RedisBroker is an implementation of the Broker interface
// that uses Redis as the message broker.
type RedisBroker struct {
	stream string
	client *redis.Client
}

func NewRedisBroker(rdb *redis.Client, opts ...Option) *RedisBroker {
	rb := &RedisBroker{
		client: rdb,
		stream: "ratebroker",
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(rb)
	}

	return rb
}

// WithStream sets the Redis stream name, a good value
// would be the name of your application.
// default: "ratebroker"
func WithStream(stream string) func(*RedisBroker) {
	return func(rb *RedisBroker) {
		rb.stream = stream
	}
}

// Publish publishes a message to a Redis stream
func (r *RedisBroker) Publish(ctx context.Context, message Message) error {
	values := map[string]interface{}{
		"broker_id": message.BrokerID,
		"event":     message.Event,
		"timestamp": message.Timestamp,
		"key":       message.Key,
	}

	return r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: r.stream,
		Values: values,
	}).Err()
}

// Consume listens to messages on a Redis stream and processes them with handlerFunc
func (r *RedisBroker) Consume(ctx context.Context, handlerFunc func(Message)) error {
	// The 'lastMessageID' is initially set to '$' for new messages.
	var lastMessageID = "$"

	for {
		// Check the context before a new loop iteration starts
		if ctx.Err() != nil {
			return ctx.Err() // Return the actual error that caused the context cancellation
		}

		// Read messages from the stream.
		// 'Count' can be adjusted based on how many messages we want to process per iteration.
		messages, err := r.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{r.stream, lastMessageID},
			Count:   10, // Define how many messages you want to retrieve at once
			Block:   0,  // Setting the block time to 0 makes the XRead command non-blocking
		}).Result()

		if err != nil {
			return err // Handle the error based on your application's requirements
		}

		// Process messages if any.
		for _, message := range messages {
			for _, xMessage := range message.Messages {
				bstr, _ := json.Marshal(xMessage.Values)

				var msg Message
				// Deserialize the message
				if err := json.Unmarshal(bstr, &msg); err != nil {
					return err // Handle deserialization error
				}

				// Call the handler function to process the message
				handlerFunc(msg)

				// Update lastMessageID to acknowledge processing.
				lastMessageID = xMessage.ID
			}
		}
	}
}
