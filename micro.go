// Package micro is a pluggable framework for microservices
package micro

import (
	"context"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/server"
)

type serviceKey struct{}

// Service is an interface that wraps the lower level libraries
// within go-micro. Its a convenience method for building
// and initialising services.
type Service interface {
	// The service name
	Name() string
	// Init initialises options
	Init(...Option)
	// Options returns the current options
	Options() Options
	// Client is used to call services
	Client() client.Client
	// Server is for handling requests and events
	Server() server.Server
	// Run the service
	Run() error
	// The service implementation
	String() string
}

// Function is a one time executing Service
type Function interface {
	// Inherits Service interface
	Service
	// Done signals to complete execution
	Done() error
	// Handle registers an RPC handler
	Handle(v interface{}) error
	// Subscribe registers a subscriber
	Subscribe(topic string, v interface{}) error
}

/*
// Type Event is a future type for acting on asynchronous events
type Event interface {
	// Publish publishes a message to the event topic
	Publish(ctx context.Context, msg interface{}, opts ...client.PublishOption) error
	// Subscribe to the event
	Subscribe(ctx context.Context, v in
}

// Resource is a future type for defining dependencies
type Resource interface {
	// Name of the resource
	Name() string
	// Type of resource
	Type() string
	// Method of creation
	Create() error
}
*/

// Event is used to publish messages to a topic
type Event interface {
	// Publish publishes a message to the event topic
	Publish(ctx context.Context, msg interface{}, opts ...client.PublishOption) error
}

// Type alias to satisfy the deprecation
type Publisher = Event
