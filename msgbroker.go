package ratebroker

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jpillora/backoff"
	"golang.org/x/exp/slog"
)

const (
	// RequestAccepted is the event type for a request that was accepted.
	RequestAccepted = "REQUEST_ACCEPTED"
)

// Message represents the structure of the data that will be sent through the broker.
type Message struct {
	BrokerID  string    `json:"broker_id"` // The ID of the broker
	Event     string    `json:"event"`     // Type of event, e.g., "request_accepted"
	Timestamp time.Time `json:"timestamp"` // When the event occurred
	Key       string    `json:"key"`       // The key of the request, e.g., IP, UserID, etc.
}

// MessageBroker is an interface that defines the methods that a broker must implement.
// The broker is responsible for publishing and consuming messages and could be implemented in
// any message broker, e.g., Redis, Kafka, etc.
type MessageBroker interface {
	Publish(ctx context.Context, msg Message) error
	Consume(ctx context.Context, handlerFunc func(Message)) error
}

// RedisMessageBroker is an implementation of the Broker interface
// that uses Redis as the message broker.
type RedisMessageBroker struct {
	stream string
	client *redis.Client

	//name time duration for pull older messages on startup
	initialLoadOffset time.Duration

	backoff *backoff.Backoff
}

func NewRedisMessageBroker(rdb *redis.Client, opts ...func(*RedisMessageBroker)) *RedisMessageBroker {
	// Create an exponential backoff configuration
	b := backoff.Backoff{
		//These are the defaults
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
		Jitter: false,
	}

	rb := &RedisMessageBroker{
		client:  rdb,
		stream:  "ratebroker",
		backoff: &b,
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
func WithStream(stream string) func(*RedisMessageBroker) {
	return func(rb *RedisMessageBroker) {
		rb.stream = stream
	}
}

// WithInitLoadOffset is a time duration that will allow
// the pulling of older messages on startup from the topic.
// This would be used for not losing client request history on
// restart.
func WithInitLoadOffset(offset time.Duration) func(*RedisMessageBroker) {
	return func(rb *RedisMessageBroker) {
		rb.initialLoadOffset = offset
	}
}

// Publish publishes a message to a Redis stream
func (r *RedisMessageBroker) Publish(ctx context.Context, message Message) error {
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
func (r *RedisMessageBroker) Consume(ctx context.Context, handlerFunc func(Message)) error {

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
			// log and implement a retry with backoff mechanism
			slog.Error("Error reading messages from stream", slog.Any("error", err))
			time.Sleep(r.backoff.Duration())
			continue
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

func (r *RedisMessageBroker) loadInitialMessageID() string {
	lastMessageID := "$"

	if r.initialLoadOffset > 0 {
		// Calculate the timestamp for one minute ago
		oneMinuteAgo := time.Now().Add(-1 * r.initialLoadOffset)

		lastMessageID = strconv.FormatInt(oneMinuteAgo.Unix(), 10)
	}

	return lastMessageID
}
