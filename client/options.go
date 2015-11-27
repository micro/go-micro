package client

import (
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/transport"
)

type options struct {
	contentType string
	broker      broker.Broker
	codecs      map[string]codec.Codec
	registry    registry.Registry
	transport   transport.Transport
	wrappers    []Wrapper
}

// Broker to be used for pub/sub
func Broker(b broker.Broker) Option {
	return func(o *options) {
		o.broker = b
	}
}

// Codec to be used to encode/decode requests for a given content type
func Codec(contentType string, c codec.Codec) Option {
	return func(o *options) {
		o.codecs[contentType] = c
	}
}

// Default content type of the client
func ContentType(ct string) Option {
	return func(o *options) {
		o.contentType = ct
	}
}

// Registry to find nodes for a given service
func Registry(r registry.Registry) Option {
	return func(o *options) {
		o.registry = r
	}
}

// Transport to use for communication e.g http, rabbitmq, etc
func Transport(t transport.Transport) Option {
	return func(o *options) {
		o.transport = t
	}
}

// Adds a Wrapper to a list of options passed into the client
func Wrap(w Wrapper) Option {
	return func(o *options) {
		o.wrappers = append(o.wrappers, w)
	}
}
