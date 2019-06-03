// Package nats provides a NATS control plane
package nats

import (
	"github.com/micro/go-micro"
	broker "github.com/micro/go-plugins/broker/nats"
	registry "github.com/micro/go-plugins/registry/nats"
	transport "github.com/micro/go-plugins/transport/nats"
)

// NewService returns a NATS micro.Service
func NewService(opts ...micro.Option) micro.Service {
	// initialise nats components
	b := broker.NewBroker()
	r := registry.NewRegistry()
	t := transport.NewTransport()

	// create new options
	options := []micro.Option{
		micro.Broker(b),
		micro.Registry(r),
		micro.Transport(t),
	}

	// append user options
	options = append(options, opts...)

	// return a nats service
	return micro.NewService(options...)
}
