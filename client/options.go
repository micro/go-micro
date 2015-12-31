package client

import (
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/selector"
	"github.com/micro/go-micro/transport"
)

type Options struct {
	ContentType string
	Broker      broker.Broker
	Codecs      map[string]codec.NewCodec
	Registry    registry.Registry
	Selector    selector.Selector
	Transport   transport.Transport
	Wrappers    []Wrapper

	// Other options to be used by client implementations
	Options map[string]string
}

type CallOptions struct {
	SelectOptions []selector.SelectOption

	// Other options to be used by client implementations
	Options map[string]string
}

type PublishOptions struct {
	// Other options to be used by client implementations
	Options map[string]string
}

type RequestOptions struct {
	Stream bool

	// Other options to be used by client implementations
	Options map[string]string
}

// Broker to be used for pub/sub
func Broker(b broker.Broker) Option {
	return func(o *Options) {
		o.Broker = b
	}
}

// Codec to be used to encode/decode requests for a given content type
func Codec(contentType string, c codec.NewCodec) Option {
	return func(o *Options) {
		o.Codecs[contentType] = c
	}
}

// Default content type of the client
func ContentType(ct string) Option {
	return func(o *Options) {
		o.ContentType = ct
	}
}

// Registry to find nodes for a given service
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// Transport to use for communication e.g http, rabbitmq, etc
func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

// Select is used to select a node to route a request to
func Selector(s selector.Selector) Option {
	return func(o *Options) {
		o.Selector = s
	}
}

// Adds a Wrapper to a list of options passed into the client
func Wrap(w Wrapper) Option {
	return func(o *Options) {
		o.Wrappers = append(o.Wrappers, w)
	}
}

// Call Options

func WithSelectOption(so selector.SelectOption) CallOption {
	return func(o *CallOptions) {
		o.SelectOptions = append(o.SelectOptions, so)
	}
}

// Request Options

func StreamingRequest() RequestOption {
	return func(o *RequestOptions) {
		o.Stream = true
	}
}
