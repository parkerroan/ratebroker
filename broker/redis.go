package broker

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// Option is a function that can be passed into NewRateBroker to configure the RateBroker.
type Option func(*RedisBroker)

// RedisBroker is an implementation of the Broker interface
// that uses Redis as the message broker.
type RedisBroker struct {
	stream string
	client *redis.Client

	//name time duration for pull older messages on startup
	initialLoadOffset time.Duration
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

// WithInitLoadOffset is a time duration that will allow
// the pulling of older messages on startup from the topic.
// This would be used for not losing client request history on
// restart.
func WithInitLoadOffset(offset time.Duration) func(*RedisBroker) {
	return func(rb *RedisBroker) {
		rb.initialLoadOffset = offset
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

	// Format it as needed for your message fetching system. Here it's formatted as Unix time, but you might need a different format.
	lastMessageID := r.loadInitialMessageID()

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
			Block:   0,
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

func (r *RedisBroker) loadInitialMessageID() string {
	lastMessageID := "$"

	if r.initialLoadOffset > 0 {
		// Calculate the timestamp for one minute ago
		oneMinuteAgo := time.Now().Add(-1 * r.initialLoadOffset)

		lastMessageID = strconv.FormatInt(oneMinuteAgo.Unix(), 10)
	}

	return lastMessageID
}
