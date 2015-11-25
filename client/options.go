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

func Broker(b broker.Broker) Option {
	return func(o *options) {
		o.broker = b
	}
}

func Codec(contentType string, cf CodecFunc) Option {
	return func(o *options) {
		o.codecs[contentType] = cf
	}
}

func ContentType(ct string) Option {
	return func(o *options) {
		o.contentType = ct
	}
}

func Registry(r registry.Registry) Option {
	return func(o *options) {
		o.registry = r
	}
}

func Transport(t transport.Transport) Option {
	return func(o *options) {
		o.transport = t
	}
}
