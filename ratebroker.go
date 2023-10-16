package ratebroker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/parkerroan/ratebroker/broker"
	"github.com/parkerroan/ratebroker/limiter"
	"golang.org/x/exp/slog"
)

type Option func(*RateBroker)

// RateLimiter is the main structure that will use a Limiter to enforce rate limits.
type RateBroker struct {
	broker      broker.Broker
	limiterFunc limiter.NewLimiterFunc
	maxRequests int
	window      time.Duration
	userCache   *ristretto.Cache
	mu          sync.Mutex
	//TODO add a cache for limiter per user
}

// NewRateBroker creates a RateLimiter with the provided Limiter.
func NewRateBroker(broker broker.Broker, limiterFunc limiter.NewLimiterFunc, opts ...Option) *RateBroker {

	rb := &RateBroker{
		limiterFunc: limiterFunc,
		broker:      broker,
		maxRequests: 30,
		window:      10 * time.Second,
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(rb)
	}

	// Create a new cache with a high size limit (adjust as needed) if one is not provided
	if rb.userCache == nil {
		cache, err := ristretto.NewCache(&ristretto.Config{
			NumCounters: 1e7,     // Num keys to track frequency of (10M).
			MaxCost:     1 << 30, // Maximum cost of cache (1GB).
			BufferItems: 64,      // Number of keys per Get buffer.
		})
		if err != nil {
			log.Fatal(err) // handle error according to your strategy
		}
		rb.userCache = cache
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

// Start ...
func (rb *RateBroker) Start(ctx context.Context) {

	go func() {
		rb.broker.Consume(ctx, rb.BrokerHandleFunc)
	}()
}

// TryAccept is a method on RateLimiter that checks a new request against the current rate limit.
func (rb *RateBroker) TryAccept(key string) bool {
	now := time.Now()
	var block bool

	if userLimit := rb.getLimiterForUser(key); userLimit != nil {
		block = !userLimit.Try(now)
	}

	if block {
		message := broker.Message{
			Event:     broker.RequestAccepted,
			Timestamp: time.Now(),
			Key:       key,
		}
		err := rb.broker.Publish(context.Background(), message)
		if err != nil {
			slog.Error("error publishing message", slog.Any("error", err.Error()))
		}
		// handle err if necessary...
	}

	return !block
}

func (rb *RateBroker) BrokerHandleFunc(message broker.Message) {
	slog.Info("message received", slog.Any("message", message))

	//TODO lookup limiter for user
	userLimit := rb.getLimiterForUser(message.Key)
	if userLimit == nil {
		userLimit = rb.limiterFunc(rb.maxRequests, rb.window)
		rb.userCache.Set(message.Key, userLimit, 1)
	}

	userLimit.Accept(message.Timestamp)
}

func (rb *RateBroker) getLimiterForUser(userID string) limiter.Limiter {
	var userLimiter limiter.Limiter

	// Try to get the limiter from cache
	item, found := rb.userCache.Get(userID)
	if !found {
		return nil
	}

	// Assert the type to your limiter interface or struct
	userLimiter = item.(limiter.Limiter)
	return userLimiter
}
