package ratebroker

import (
	"context"
	"time"

	"github.com/parkerroan/ratebroker/broker"
	"github.com/parkerroan/ratebroker/limiter"
	"golang.org/x/exp/slog"
)

type Option func(*RateBroker)

// RateLimiter is the main structure that will use a Limiter to enforce rate limits.
type RateBroker struct {
	broker      broker.Broker
	limiter     limiter.Limiter
	maxRequests int
	window      time.Duration
	//TODO add a cache for limiter per user
}

// NewRateBroker creates a RateLimiter with the provided Limiter.
func NewRateBroker(broker broker.Broker, limiter limiter.Limiter, opts ...Option) *RateBroker {
	rb := &RateBroker{
		limiter:     limiter,
		broker:      broker,
		maxRequests: 30,
		window:      10 * time.Second,
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(rb)
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
func (rl *RateBroker) Start(ctx context.Context) {

	go func() {
		rl.broker.Consume(ctx, rl.BrokerHandleFunc)
	}()
}

// TryAccept is a method on RateLimiter that checks a new request against the current rate limit.
func (rl *RateBroker) TryAccept() bool {

	now := time.Now()
	allowed := rl.limiter.Allowed(now)

	if allowed {
		message := broker.Message{
			Event:     broker.RequestAccepted,
			Timestamp: time.Now(),
			// populate other fields as necessary...
		}
		err := rl.broker.Publish(context.Background(), message)
		if err != nil {
			slog.Error("error publishing message", slog.Any("error", err.Error()))
		}
		// handle err if necessary...
	}

	return allowed
}

func (rl *RateBroker) BrokerHandleFunc(message broker.Message) {
	slog.Info("message received", slog.Any("message", message))

	//TODO lookup limiter for user

	rl.limiter.Accept(message.Timestamp)
}
