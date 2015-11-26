package client

import (
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/transport"
)

type options struct {
	contentType string
	codecs      map[string]CodecFunc
	broker      broker.Broker
	registry    registry.Registry
	transport   transport.Transport
}

// Broker to be used for pub/sub
func Broker(b broker.Broker) Option {
	return func(o *options) {
		o.broker = b
	}
}

// Codec to be used to encode/decode requests for a given content type
func Codec(contentType string, cf CodecFunc) Option {
	return func(o *options) {
		o.codecs[contentType] = cf
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
