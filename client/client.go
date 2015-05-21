package client

import (
	"github.com/myodc/go-micro/transport"
)

type Client interface {
	NewRequest(string, string, interface{}) Request
	NewProtoRequest(string, string, interface{}) Request
	NewJsonRequest(string, string, interface{}) Request
	Call(Request, interface{}) error
	CallRemote(string, string, Request, interface{}) error
}

type options struct {
	transport transport.Transport
}

type Option func(*options)

var (
	DefaultClient Client = NewRpcClient()
)

func Transport(t transport.Transport) Option {
	return func(o *options) {
		o.transport = t
	}
}

func Call(request Request, response interface{}) error {
	return DefaultClient.Call(request, response)
}

func CallRemote(address, path string, request Request, response interface{}) error {
	return DefaultClient.CallRemote(address, path, request, response)
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
