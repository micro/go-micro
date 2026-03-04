// Package micro is a pluggable framework for microservices
package micro

import (
	"context"

	"go-micro.dev/v5/client"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/service"
)

type serviceKey struct{}

// Service is an interface that wraps the lower level libraries
// within go-micro. Its a convenience method for building
// and initializing services.
type Service interface {
	// The service name
	Name() string
	// Init initializes options
	Init(...Option)
	// Options returns the current options
	Options() Options
	// Register the handler
	Handle(v interface{}) error
	// Client is used to call services
	Client() client.Client
	// Server is for handling requests and events
	Server() server.Server
	// Start the service
	Start() error
	// Stop the service
	Stop() error
	// Run the service (start, block on signal, then stop)
	Run() error
	// The service implementation
	String() string
}

// Group is a set of services that share lifecycle management.
type Group = service.Group

type Option = service.Option

type Options = service.Options

// Event is used to publish messages to a topic.
type Event interface {
	// Publish publishes a message to the event topic
	Publish(ctx context.Context, msg interface{}, opts ...client.PublishOption) error
}

// Type alias to satisfy the deprecation.
type Publisher = Event

// New represents the new service
func New(name string) Service {
	return NewService(
		service.Name(name),
	)
}

// NewService creates and returns a new Service based on the packages within.
func NewService(opts ...Option) Service {
	return service.New(opts...)
}

// NewGroup creates a service group for running multiple services
// in a single binary with shared lifecycle management.
func NewGroup(svcs ...Service) *Group {
	var ss []*service.ServiceImpl
	for _, s := range svcs {
		if si, ok := s.(*service.ServiceImpl); ok {
			ss = append(ss, si)
		}
	}
	return service.NewGroup(ss...)
}

// FromContext retrieves a Service from the Context.
func FromContext(ctx context.Context) (Service, bool) {
	s, ok := ctx.Value(serviceKey{}).(Service)
	return s, ok
}

// NewContext returns a new Context with the Service embedded within it.
func NewContext(ctx context.Context, s Service) context.Context {
	return context.WithValue(ctx, serviceKey{}, s)
}

// NewEvent creates a new event publisher.
func NewEvent(topic string, c client.Client) Event {
	if c == nil {
		c = client.NewClient()
	}

	return &event{c, topic}
}

// RegisterHandler is syntactic sugar for registering a handler.
func RegisterHandler(s server.Server, h interface{}, opts ...server.HandlerOption) error {
	return s.Handle(s.NewHandler(h, opts...))
}

// RegisterSubscriber is syntactic sugar for registering a subscriber.
func RegisterSubscriber(topic string, s server.Server, h interface{}, opts ...server.SubscriberOption) error {
	return s.Subscribe(s.NewSubscriber(topic, h, opts...))
}
