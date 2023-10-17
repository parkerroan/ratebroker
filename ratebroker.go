package ratebroker

import (
	"context"
	"log"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/google/uuid"
	"github.com/parkerroan/ratebroker/broker"
	"github.com/parkerroan/ratebroker/limiter"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/semaphore"
)

// Option is a function that can be passed into NewRateBroker to configure the RateBroker.
type Option func(*RateBroker)

// NewLimiterFunc is a function that creates a new limiter.
type NewLimiterFunc func(int, time.Duration) limiter.Limiter

// RateBroker is the main structure that will use a Limiter to enforce rate limits.
type RateBroker struct {
	id             string
	broker         broker.Broker
	newLimiterFunc NewLimiterFunc
	maxRequests    int
	window         time.Duration
	cache          *ristretto.Cache
	maxThreads     int
	sem            *semaphore.Weighted
}

// NewRateBroker creates a RateLimiter with the provided Limiter.
func NewRateBroker(opts ...Option) *RateBroker {

	//Defaults
	rb := &RateBroker{
		id:             uuid.NewString(),
		newLimiterFunc: limiter.NewRingLimiterConstructorFunc(),
		broker:         nil,
		maxRequests:    30,
		window:         10 * time.Second,
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(rb)
	}

	// Create a new cache with a high size limit (adjust as needed) if one is not provided
	if rb.cache == nil {
		cache, err := ristretto.NewCache(&ristretto.Config{
			NumCounters: 1e7,     // Num keys to track frequency of (10M).
			MaxCost:     1 << 30, // Maximum cost of cache (1GB).
			BufferItems: 64,      // Number of keys per Get buffer.
		})
		if err != nil {
			log.Fatal(err) // handle error according to your strategy
		}
		rb.cache = cache
	}

	return rb
}

// WithBroker sets the broker for the RateBroker.
//
// If no broker is provided, the RateBroker will not publish events
// to the message broker.
//
// Instead it will use the local limiter to enforce rate limits without distribution.
func WithBroker(broker broker.Broker) Option {
	return func(rb *RateBroker) {
		rb.broker = broker
	}
}

// WithID sets the ID for the RateBroker.
func WithID(id string) Option {
	return func(rb *RateBroker) {
		rb.id = id
	}
}

// WithMaxRequests sets the maximum number of requests allowed for the RateBroker.
func WithMaxRequests(max int) Option {
	return func(rb *RateBroker) {
		rb.maxRequests = max
	}
}

// WithMaxThreads sets the maximum number of threads allowed for publishing events
// to the message broker.
func WithMaxThreads(maxThreads int) Option {
	return func(rb *RateBroker) {
		rb.maxThreads = maxThreads
		rb.sem = semaphore.NewWeighted(int64(maxThreads))
	}
}

// WithWindow sets the time window for the RateBroker.
func WithWindow(window time.Duration) Option {
	return func(rb *RateBroker) {
		rb.window = window
	}
}

// WithLimiterContructorFunc sets the function used to create a new limiter.
func WithLimiterContructorFunc(limiterFunc NewLimiterFunc) Option {
	return func(rb *RateBroker) {
		rb.newLimiterFunc = limiterFunc
	}
}

// Start is a method on RateLimiter that starts the broker consuming messages
// and handling them in the background.
func (rb *RateBroker) Start(ctx context.Context) {
	if rb.broker == nil {
		go func() {
			err := rb.broker.Consume(ctx, rb.brokerHandleFunc)
			if err != nil {
				slog.Error("error consuming messages", slog.Any("error", err.Error()))
			}
		}()
	}
}

// TryAccept is a method on RateLimiter that checks a new request against the current rate limit.
func (rb *RateBroker) TryAccept(key string) bool {
	now := time.Now()

	var userLimit limiter.Limiter
	if userLimit = rb.getLimiter(key); userLimit == nil {
		userLimit = rb.newLimiterFunc(rb.maxRequests, rb.window)
		rb.cache.Set(key, userLimit, 1)
	}

	if allow := userLimit.TryAccept(now); !allow {
		return false
	}

	if rb.broker != nil {
		message := broker.Message{
			BrokerID:  rb.id,
			Event:     broker.RequestAccepted,
			Timestamp: now,
			Key:       key,
		}

		err := rb.broker.Publish(context.Background(), message)
		if err != nil {
			slog.Error("error publishing message", slog.Any("error", err.Error()))
		}
	}

	return true
}

func (rb *RateBroker) publishEvent(ctx context.Context, msg broker.Message) error {
	var deferFunc func()
	if rb.sem != nil {
		deferFunc = func() {
			rb.sem.Release(1)
		}
		if err := rb.sem.Acquire(ctx, 1); err != nil {
			slog.Error("Failed to acquire semaphore", slog.Any("error", err.Error()))
			return err
		}
	}

	go func() {
		defer deferFunc()
		err := rb.broker.Publish(context.Background(), msg)
		if err != nil {
			slog.Error("error publishing message", slog.Any("error", err.Error()))
		}
	}()

	return nil
}

// BrokerHandleFunc is passed into the broker to handle incoming messages
func (rb *RateBroker) brokerHandleFunc(message broker.Message) {
	slog.Info("message received", slog.Any("message", message))

	//return early as we don't want to process our own messages
	if message.BrokerID == rb.id {
		return
	}

	limit := rb.getLimiter(message.Key)
	if limit == nil {
		limit = rb.newLimiterFunc(rb.maxRequests, rb.window)
		rb.cache.Set(message.Key, limit, 1)
	}

	limit.Accept(message.Timestamp)

}

func (rb *RateBroker) getLimiter(key string) limiter.Limiter {
	var userLimiter limiter.Limiter

	// Try to get the limiter from cache
	item, found := rb.cache.Get(key)
	if !found {
		return nil
	}

	// Assert the type to your limiter interface or struct
	userLimiter = item.(limiter.Limiter)
	return userLimiter
}
