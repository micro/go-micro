package client

import (
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/selector"
	"github.com/micro/go-micro/transport"
)

type options struct {
	contentType string
	broker      broker.Broker
	codecs      map[string]codec.NewCodec
	registry    registry.Registry
	selector    selector.Selector
	transport   transport.Transport
	wrappers    []Wrapper
}

type callOptions struct {
	selectOptions []selector.SelectOption
}

type publishOptions struct{}

type requestOptions struct {
	stream bool
}

// Broker to be used for pub/sub
func Broker(b broker.Broker) Option {
	return func(o *options) {
		o.broker = b
	}
}

// Codec to be used to encode/decode requests for a given content type
func Codec(contentType string, c codec.NewCodec) Option {
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

// Select is used to select a node to route a request to
func Selector(s selector.Selector) Option {
	return func(o *options) {
		o.selector = s
	}
}

// Adds a Wrapper to a list of options passed into the client
func Wrap(w Wrapper) Option {
	return func(o *options) {
		o.wrappers = append(o.wrappers, w)
	}
}

// Call Options

func WithSelectOption(so selector.SelectOption) CallOption {
	return func(o *callOptions) {
		o.selectOptions = append(o.selectOptions, so)
	}
}

// Request Options

func StreamingRequest() RequestOption {
	return func(o *requestOptions) {
		o.stream = true
	}
}
