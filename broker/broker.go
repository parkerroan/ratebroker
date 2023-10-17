package broker

import (
	"context"
	"time"
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

// Limiter is the interface that abstracts the limitations functionality.
type Broker interface {
	Publish(ctx context.Context, msg Message) error
	Consume(ctx context.Context, handlerFunc func(Message)) error
}
