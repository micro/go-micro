package server

import (
	"github.com/micro/go-micro/registry"
)

// Handler interface represents a Service request handler. It's generated
// by passing any type of public concrete object with methods into server.NewHandler.
// Most will pass in a struct.
//
// Example:
//
//	type Service struct {}
//
//	func (s *Service) Method(context, request, response) error {
//		return nil
//	}
//
type Handler interface {
	Name() string
	Handler() interface{}
	Endpoints() []*registry.Endpoint
	Options() HandlerOptions
}

// Subscriber interface represents a subscription to a given topic using
// a specific subscriber function or object with methods.
type Subscriber interface {
	Topic() string
	Subscriber() interface{}
	Endpoints() []*registry.Endpoint
	Options() SubscriberOptions
}

type HandlerOptions struct {
	Internal bool
	Metadata map[string]map[string]string
}

type SubscriberOptions struct {
	Queue    string
	Internal bool
}

// EndpointMetadata is a Handler option that allows metadata to be added to
// individual endpoints.
func EndpointMetadata(name string, md map[string]string) HandlerOption {
	return func(o *HandlerOptions) {
		o.Metadata[name] = md
	}
}

// Internal Handler options specifies that a handler is not advertised
// to the discovery system. In the future this may also limit request
// to the internal network or authorised user.
func InternalHandler(b bool) HandlerOption {
	return func(o *HandlerOptions) {
		o.Internal = b
	}
}

// Internal Subscriber options specifies that a subscriber is not advertised
// to the discovery system.
func InternalSubscriber(b bool) SubscriberOption {
	return func(o *SubscriberOptions) {
		o.Internal = b
	}
}

// Shared queue name distributed messages across subscribers
func SubscriberQueue(n string) SubscriberOption {
	return func(o *SubscriberOptions) {
		o.Queue = n
	}
}
