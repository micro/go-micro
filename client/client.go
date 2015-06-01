package client

import (
	"github.com/myodc/go-micro/registry"
	"github.com/myodc/go-micro/transport"
	"golang.org/x/net/context"
)

type Client interface {
	NewRequest(string, string, interface{}) Request
	NewProtoRequest(string, string, interface{}) Request
	NewJsonRequest(string, string, interface{}) Request
	Call(context.Context, Request, interface{}) error
	CallRemote(context.Context, string, Request, interface{}) error
	Stream(context.Context, Request, interface{}) (Streamer, error)
	StreamRemote(context.Context, string, Request, interface{}) (Streamer, error)
}

type Streamer interface {
	Request() Request
	Error() error
	Close() error
}

type options struct {
	registry  registry.Registry
	transport transport.Transport
}

type Option func(*options)

var (
	DefaultClient Client = newRpcClient()
)

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

func Call(ctx context.Context, request Request, response interface{}) error {
	return DefaultClient.Call(ctx, request, response)
}

func CallRemote(ctx context.Context, address string, request Request, response interface{}) error {
	return DefaultClient.CallRemote(ctx, address, request, response)
}

func Stream(ctx context.Context, request Request, responseChan interface{}) (Streamer, error) {
	return DefaultClient.Stream(ctx, request, responseChan)
}

func StreamRemote(ctx context.Context, address string, request Request, responseChan interface{}) (Streamer, error) {
	return DefaultClient.StreamRemote(ctx, address, request, responseChan)
}

func NewClient(opt ...Option) Client {
	return newRpcClient(opt...)
}

func NewRequest(service, method string, request interface{}) Request {
	return DefaultClient.NewRequest(service, method, request)
}

func NewProtoRequest(service, method string, request interface{}) Request {
	return DefaultClient.NewProtoRequest(service, method, request)
}

func NewJsonRequest(service, method string, request interface{}) Request {
	return DefaultClient.NewJsonRequest(service, method, request)
}
