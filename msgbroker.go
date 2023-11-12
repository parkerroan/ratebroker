package ratebroker

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/jpillora/backoff"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/semaphore"
)

const (
	// RequestAccepted is the event type for a request that was accepted.
	RequestAccepted = "REQUEST_ACCEPTED"
)

// RateMessage represents the structure of the data that will be sent through the broker.
type RateMessage struct {
	Events []RateEvent `json:"events"`
}

// RateEvent represents the structure of the data that will be sent through the broker.
type RateEvent struct {
	BrokerID  string    `json:"broker_id"` // The ID of the broker
	Event     string    `json:"event"`     // Type of event, e.g., "request_accepted"
	Timestamp time.Time `json:"timestamp"` // When the event occurred
	Key       string    `json:"key"`       // The key of the request, e.g., IP, UserID, etc.
}

func (e RateEvent) MarshalBinary() ([]byte, error) {
	return json.Marshal(e)
}

// MessageBroker is an interface that defines the methods that a broker must implement.
// The broker is responsible for publishing and consuming messages and could be implemented in
// any message broker, e.g., Redis, Kafka, etc.
type MessageBroker interface {
	Start(ctx context.Context, handlerFunc func(RateEvent))
	Publish(ctx context.Context, msg RateEvent) error
}

// RedisMessageBroker is an implementation of the Broker interface
// that uses Redis as the message broker.
type RedisMessageBroker struct {
	stream string
	client *redis.Client

	//name time duration for pull older messages on startup
	initialLoadOffset time.Duration
	maxStreamLen      int64

	backoff        *backoff.Backoff
	publishChannel chan RateEvent

	sem *semaphore.Weighted
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
		client:         rdb,
		stream:         "ratebroker",
		backoff:        &b,
		publishChannel: make(chan RateEvent, 100),
		sem:            semaphore.NewWeighted(int64(100)), // default to 100 publish threads
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(rb)
	}

	return rb
}

// WithMaxThreads sets the maximum number of publish threads allowed for publishing events
// to the message broker.
func WithMaxThreads(maxThreads int) func(*RedisMessageBroker) {
	return func(rb *RedisMessageBroker) {
		rb.sem = semaphore.NewWeighted(int64(maxThreads))
	}
}

// WithStream sets the Redis stream name, a good value
// would be the name of your application.
// default: "ratebroker"
func WithStream(stream string) func(*RedisMessageBroker) {
	return func(rb *RedisMessageBroker) {
		rb.stream = stream
	}
}

// WithCappedStream sets the Redis stream max length.
func WithCappedStream(maxLen int64) func(*RedisMessageBroker) {
	return func(rb *RedisMessageBroker) {
		rb.maxStreamLen = maxLen
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

// Start is a method on RateLimiter that starts the broker consuming messages
// and handling them in the background.
func (r *RedisMessageBroker) Start(ctx context.Context, handlerFunc func(RateEvent)) {
	go func() {
		err := r.StartPublisher(ctx)
		if err != nil {
			slog.Error("error publishing messages in publisher", slog.Any("error", err.Error()))
		}
	}()

	go func() {
		err := r.Consume(ctx, handlerFunc)
		if err != nil {
			slog.Error("error consuming messages", slog.Any("error", err.Error()))
		}
	}()

}

// StartPublisher publishes a message to a Redis stream from a channel
func (r *RedisMessageBroker) StartPublisher(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			batchSize := 100
			events := make([]RateEvent, 0, batchSize)

			// Block until we receive the first message
			select {
			case event := <-r.publishChannel:
				events = append(events, event)
			case <-ctx.Done():
				return nil
			}

			// Try to gather a batch of messages
			for i := 0; i < batchSize; i++ {
				select {
				case event := <-r.publishChannel:
					events = append(events, event)
				case <-ctx.Done():
					return nil
				default:
					// No more messages ready to pull off the channel, stop the loop
					i = batchSize // This will break the 'for' loop, as it makes the condition false.
				}
			}

			msg := RateMessage{
				Events: events,
			}

			deferFunc := func() {}
			if r.sem != nil {
				deferFunc = func() {
					r.sem.Release(1)
				}
				if err := r.sem.Acquire(ctx, 1); err != nil {
					slog.Error("Failed to acquire semaphore", slog.Any("error", err.Error()))
					return err
				}
			}

			go func(msg RateMessage) {
				publishCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond) // Set your own timeout duration
				defer cancel()
				defer deferFunc()
				// Publish the batch message
				if err := r.publish(publishCtx, msg); err != nil {
					slog.Error("error publishing message to redis", slog.Any("error", err.Error()))
				}
			}(msg)
		}
	}
}

func (r *RedisMessageBroker) Publish(ctx context.Context, msg RateEvent) error {
	select {
	case r.publishChannel <- msg:
		return nil
	case <-ctx.Done():
		close(r.publishChannel)
		if ctx.Err() == context.Canceled {
			slog.Info("Context was cancelled, cleaning up")
			return nil // Return nil if the context was cancelled
		}
		return ctx.Err()
	}
}

// Consume listens to messages on a Redis stream and processes them with handlerFunc
func (r *RedisMessageBroker) Consume(ctx context.Context, handlerFunc func(RateEvent)) error {

	// Format it as needed for your message fetching system. Here it's formatted as Unix time, but you might need a different format.
	lastMessageID := r.loadInitialMessageID()

	for {
		// Check the context before a new loop iteration starts
		if ctx.Err() != nil {
			if ctx.Err() == context.Canceled {
				slog.Info("Context was cancelled, cleaning up")
				return nil // Return nil if the context was cancelled
			}
			return ctx.Err() // Return the actual error that caused the context cancellation
		}

		// Read messages from the stream.
		// 'Count' can be adjusted based on how many messages we want to process per iteration.
		messages, err := r.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{r.stream, lastMessageID},
			Count:   100, // Define how many messages you want to retrieve at once
			Block:   0,
		}).Result()

		if err != nil {
			if errors.Is(err, context.Canceled) {
				slog.Info("Context was cancelled, cleaning up")
				return nil // Return nil if the context was cancelled
			}
			// log and implement a retry with backoff mechanism
			slog.Error("Error reading messages from stream", slog.Any("error", err))
			time.Sleep(r.backoff.Duration())
			continue
		}

		// setup a wait group to wait for all messages to be processed
		// before moving on to the next iteration of x-100 routines
		var wg sync.WaitGroup
		// Process messages if any.
		for _, message := range messages {
			for _, xMessage := range message.Messages {
				events := xMessage.Values["events"].(string)

				// Deserialize the message
				msg := RateMessage{}
				if err := json.Unmarshal([]byte(events), &msg.Events); err != nil {
					return err // Handle deserialization error
				}

				for _, event := range msg.Events {
					// Call the handler function to process the message
					wg.Add(1)
					go func(event RateEvent) {
						defer wg.Done()
						handlerFunc(event)
					}(event)
					// Update lastMessageID to acknowledge processing.
					lastMessageID = xMessage.ID
				}
			}
		}
		wg.Wait()
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

func (r *RedisMessageBroker) publish(ctx context.Context, msg RateMessage) error {
	eventBtyes, _ := json.Marshal(msg.Events)

	values := map[string]interface{}{
		"events": eventBtyes,
	}

	return r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: r.stream,
		Values: values,
		MaxLen: r.maxStreamLen,
		Approx: true,
	}).Err()
}
