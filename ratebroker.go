package ratebroker

import (
	"context"
	"time"

	"github.com/beevik/ntp"
	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/parkerroan/ratebroker/limiter"
	"golang.org/x/exp/slog"
)

// Option is a function that can be passed into NewRateBroker to configure the RateBroker.
type Option func(*RateBroker)

// NewLimiterFunc is a function that creates a new limiter.
type NewLimiterFunc func(int, time.Duration) limiter.Limiter

// LimitDetails is a struct that contains the max requests and window for a single limiter.
type LimitDetails struct {
	MaxRequests int
	Window      time.Duration
}

// RateBroker is the main structure that will use a Limiter to enforce rate limits.
type RateBroker struct {
	id             string
	broker         MessageBroker
	newLimiterFunc NewLimiterFunc
	maxRequests    int
	window         time.Duration
	cache          *expirable.LRU[string, limiter.Limiter]
	ntpClient      *ntp.Response // add an NTP client field
	ntpServer      string
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
		cache := expirable.NewLRU[string, limiter.Limiter](1000000, nil, 60*time.Minute)
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
func WithBroker(broker MessageBroker) Option {
	return func(rb *RateBroker) {
		rb.broker = broker
	}
}

// WithID sets the ID for the RateBroker
// ID is used to identify messages published by the
// RateBroker and should be unique per pod/replica
func WithID(id string) Option {
	return func(rb *RateBroker) {
		rb.id = id
	}
}

// WithMaxRequests sets the maximum number of requests allowed
// within the supplied window for the RateBroker.
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

// WithNTPServer is an option to set the NTP server for the RateBroker. If this option is not used,
// the RateBroker will use the system's local time.
func WithNTPServer(server string) Option {
	return func(rb *RateBroker) {
		rb.ntpServer = server
	}
}

// WithLimiterContructorFunc sets the function used to create a new limiter.
// The default is limiter.NewRingLimiterConstructorFunc()
// If you want to use a different limiter, you can pass in a function that creates it.
// For example, limiter.NewHeapLimiterConstructorFunc() is a supplied limiter
// that uses a heap instead of a ring buffer.
func WithLimiterContructorFunc(limiterFunc NewLimiterFunc) Option {
	return func(rb *RateBroker) {
		rb.newLimiterFunc = limiterFunc
	}
}

// Start is a method on RateLimiter that starts the broker consuming messages
// and handling them in the background.
func (rb *RateBroker) Start(ctx context.Context) {
	if rb.broker == nil {
		slog.Info("no broker configured, ignoring start")
		return
	}

	rb.broker.Start(ctx, rb.brokerHandleFunc)
}

// Now tries to get the time from the NTP server if available; otherwise, it uses the local time.
func (rb *RateBroker) Now() time.Time {
	if rb.ntpServer != "" {
		// Check if we need to (re)fetch the NTP time
		if rb.ntpClient == nil || time.Since(rb.ntpClient.Time) > 1*time.Minute { //re-fetch every minute
			response, err := ntp.Query(rb.ntpServer)
			if err != nil {
				slog.Error("error querying NTP server", slog.Any("error", err.Error()))
				return time.Now()
			}
			rb.ntpClient = response
		}
		return time.Now().Add(rb.ntpClient.ClockOffset)
	}

	// No NTP server configured, return system local time
	return time.Now()
}

// TryAccept is a method on RateLimiter that checks a new request against the current rate limit.
func (rb *RateBroker) TryAccept(ctx context.Context, key string) bool {
	now := rb.Now()

	userLimit := rb.getUserLimit(key)

	if allow := userLimit.TryAccept(now); !allow {
		return false
	}

	if rb.broker != nil {
		message := RateEvent{
			BrokerID:  rb.id,
			Event:     RequestAccepted,
			Timestamp: now,
			Key:       key,
		}

		err := rb.publishEvent(ctx, message)
		if err != nil {
			slog.Error("TryAccept error publishing message", slog.Any("error", err.Error()))
		}
	}

	return true
}

// TryAcceptWithInfo is a method on RateLimiter that checks a new request against the current rate limit.
func (rb *RateBroker) TryAcceptWithInfo(ctx context.Context, key string) (bool, limiter.RateLimitInfo) {
	now := rb.Now()

	userLimit := rb.getUserLimit(key)

	allow, info := userLimit.TryAcceptWithInfo(now)
	if !allow {
		return false, info
	}

	if rb.broker != nil {
		message := RateEvent{
			BrokerID:  rb.id,
			Event:     RequestAccepted,
			Timestamp: now,
			Key:       key,
		}

		err := rb.publishEvent(ctx, message)
		if err != nil {
			slog.Error("TryAccept error publishing message", slog.Any("error", err.Error()))
		}
	}

	return true, info
}

func (rb *RateBroker) getUserLimit(key string) limiter.Limiter {
	var userLimit limiter.Limiter
	if userLimit = rb.getLimiter(key); userLimit == nil {
		userLimit = rb.newLimiterFunc(rb.maxRequests, rb.window)
		rb.cache.Add(key, userLimit)
	}
	return userLimit
}

func (rb *RateBroker) publishEvent(ctx context.Context, msg RateEvent) error {
	publishCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second) // Set your own timeout duration
	defer cancel()

	err := rb.broker.Publish(publishCtx, msg)
	if err != nil {
		return err
	}

	return nil
}

// BrokerHandleFunc is passed into the broker to handle incoming messages
func (rb *RateBroker) brokerHandleFunc(message RateEvent) {
	slog.Debug("message received", slog.Any("message", message))

	//return early as we don't want to process our own messages
	if message.BrokerID == rb.id {
		return
	}

	if message.Timestamp.Before(rb.Now().Add(-1 * rb.window)) {
		slog.Warn("message too old, ignoring", slog.Any("message", message))
		return
	}

	limit := rb.getLimiter(message.Key)
	if limit == nil {
		limit = rb.newLimiterFunc(rb.maxRequests, rb.window)
		rb.cache.Add(message.Key, limit)
	}

	limit.Accept(message.Timestamp)
}

func (rb *RateBroker) getLimiter(key string) limiter.Limiter {
	// Try to get the limiter from cache
	limiter, found := rb.cache.Get(key)
	if !found {
		return nil
	}

	return limiter
}
