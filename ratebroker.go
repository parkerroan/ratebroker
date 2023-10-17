package ratebroker

import (
	"context"
	"log"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/parkerroan/ratebroker/broker"
	"github.com/parkerroan/ratebroker/limiter"
	"golang.org/x/exp/slog"
)

// Option is a function that can be passed into NewRateBroker to configure the RateBroker.
// Can also be chained together.
type Option func(*RateBroker)

// RateBroker is the main structure that will use a Limiter to enforce rate limits.
type RateBroker struct {
	broker         broker.Broker
	newLimiterFunc limiter.NewLimiterFunc
	maxRequests    int
	window         time.Duration
	cache          *ristretto.Cache
}

// NewRateBroker creates a RateLimiter with the provided Limiter.
func NewRateBroker(broker broker.Broker, newLimiterFunc limiter.NewLimiterFunc, opts ...Option) *RateBroker {

	rb := &RateBroker{
		newLimiterFunc: newLimiterFunc,
		broker:         broker,
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

// WithMaxRequests sets the maximum number of requests allowed for the RateBroker.
func WithMaxRequests(max int) Option {
	return func(rb *RateBroker) {
		rb.maxRequests = max
	}
}

// WithWindow sets the time window for the RateBroker.
func WithWindow(window time.Duration) Option {
	return func(rb *RateBroker) {
		rb.window = window
	}
}

// Start is a method on RateLimiter that starts the broker consuming messages
// and handling them in the background.
func (rb *RateBroker) Start(ctx context.Context) {
	go func() {
		err := rb.broker.Consume(ctx, rb.brokerHandleFunc)
		if err != nil {
			slog.Error("error consuming messages", slog.Any("error", err.Error()))
		}
	}()
}

// TryAccept is a method on RateLimiter that checks a new request against the current rate limit.
func (rb *RateBroker) TryAccept(key string) bool {
	now := time.Now()

	if userLimit := rb.getLimiter(key); userLimit != nil {
		if allow := userLimit.Try(now); !allow {
			return false
		}
	}

	message := broker.Message{
		Event:     broker.RequestAccepted,
		Timestamp: time.Now(),
		Key:       key,
	}

	err := rb.broker.Publish(context.Background(), message)
	if err != nil {
		slog.Error("error publishing message", slog.Any("error", err.Error()))
	}

	return true
}

// BrokerHandleFunc is passed into the broker to handle incoming messages
func (rb *RateBroker) brokerHandleFunc(message broker.Message) {
	slog.Info("message received", slog.Any("message", message))

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
