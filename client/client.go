package client

import (
	"golang.org/x/net/context"
)

type Client interface {
	NewPublication(topic string, msg interface{}) Publication
	NewRequest(service, method string, req interface{}) Request
	NewProtoRequest(service, method string, req interface{}) Request
	NewJsonRequest(service, method string, req interface{}) Request
	Call(ctx context.Context, req Request, rsp interface{}) error
	CallRemote(ctx context.Context, addr string, req Request, rsp interface{}) error
	Stream(ctx context.Context, req Request, rspChan interface{}) (Streamer, error)
	StreamRemote(ctx context.Context, addr string, req Request, rspChan interface{}) (Streamer, error)
	Publish(ctx context.Context, p Publication) error
}

type Publication interface {
	Topic() string
	Message() interface{}
	ContentType() string
}

type Request interface {
	Service() string
	Method() string
	ContentType() string
	Request() interface{}
}

type Streamer interface {
	Request() Request
	Error() error
	Close() error
}

type Option func(*options)

var (
	DefaultClient Client = newRpcClient()
)

// Makes a synchronous call to a service using the default client
func Call(ctx context.Context, request Request, response interface{}) error {
	return DefaultClient.Call(ctx, request, response)
}

// Makes a synchronous call to the specified address using the default client
func CallRemote(ctx context.Context, address string, request Request, response interface{}) error {
	return DefaultClient.CallRemote(ctx, address, request, response)
}

// Creates a streaming connection with a service and returns responses on the
// channel passed in. It's upto the user to close the streamer.
func Stream(ctx context.Context, request Request, responseChan interface{}) (Streamer, error) {
	return DefaultClient.Stream(ctx, request, responseChan)
}

// Creates a streaming connection to the address specified.
func StreamRemote(ctx context.Context, address string, request Request, responseChan interface{}) (Streamer, error) {
	return DefaultClient.StreamRemote(ctx, address, request, responseChan)
}

// Publishes a publication using the default client. Using the underlying broker
// set within the options.
func Publish(ctx context.Context, p Publication) error {
	return DefaultClient.Publish(ctx, p)
}

// Creates a new client with the options passed in
func NewClient(opt ...Option) Client {
	return newRpcClient(opt...)
}

// Creates a new publication using the default client
func NewPublication(topic string, message interface{}) Publication {
	return DefaultClient.NewPublication(topic, message)
}

// Creates a new request using the default client. Content Type will
// be set to the default within options and use the appropriate codec
func NewRequest(service, method string, request interface{}) Request {
	return DefaultClient.NewRequest(service, method, request)
}

// Creates a new protobuf request using the default client
func NewProtoRequest(service, method string, request interface{}) Request {
	return DefaultClient.NewProtoRequest(service, method, request)
}

// Creates a new json request using the default client
func NewJsonRequest(service, method string, request interface{}) Request {
	return DefaultClient.NewJsonRequest(service, method, request)
}
