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
}

// Subscriber interface represents a subscription to a given topic using
// a specific subscriber function or object with methods.
type Subscriber interface {
	Topic() string
	Subscriber() interface{}
	Endpoints() []*registry.Endpoint
}
