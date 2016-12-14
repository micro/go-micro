// Package client is an interface for an RPC client
package client

import (
	"time"

	"golang.org/x/net/context"
)

// Client is the interface used to make requests to services.
// It supports Request/Response via Transport and Publishing via the Broker.
// It also supports bidiectional streaming of requests.
type Client interface {
	Init(...Option) error
	Options() Options
	NewPublication(topic string, msg interface{}) Publication
	NewRequest(service, method string, req interface{}, reqOpts ...RequestOption) Request
	NewProtoRequest(service, method string, req interface{}, reqOpts ...RequestOption) Request
	NewJsonRequest(service, method string, req interface{}, reqOpts ...RequestOption) Request
	Call(ctx context.Context, req Request, rsp interface{}, opts ...CallOption) error
	CallRemote(ctx context.Context, addr string, req Request, rsp interface{}, opts ...CallOption) error
	Stream(ctx context.Context, req Request, opts ...CallOption) (Streamer, error)
	StreamRemote(ctx context.Context, addr string, req Request, opts ...CallOption) (Streamer, error)
	Publish(ctx context.Context, p Publication, opts ...PublishOption) error
	String() string
}

// Publication is the interface for a message published asynchronously
type Publication interface {
	Topic() string
	Message() interface{}
	ContentType() string
}

// Request is the interface for a synchronous request used by Call or Stream
type Request interface {
	Service() string
	Method() string
	ContentType() string
	Request() interface{}
	// indicates whether the request will be a streaming one rather than unary
	Stream() bool
}

// Streamer is the inteface for a bidirectional synchronous stream
type Streamer interface {
	Context() context.Context
	Request() Request
	Send(interface{}) error
	Recv(interface{}) error
	Error() error
	Close() error
}

// Option used by the Client
type Option func(*Options)

// CallOption used by Call or Stream
type CallOption func(*CallOptions)

// PublishOption used by Publish
type PublishOption func(*PublishOptions)

// RequestOption used by NewRequest
type RequestOption func(*RequestOptions)

var (
	// DefaultClient is a default client to use out of the box
	DefaultClient Client = newRpcClient()
	// DefaultBackoff is the default backoff function for retries
	DefaultBackoff = exponentialBackoff
	// DefaultRetry is the default check-for-retry function for retries
	DefaultRetry = alwaysRetry
	// DefaultRetries is the default number of times a request is tried
	DefaultRetries = 1
	// DefaultRequestTimeout is the default request timeout
	DefaultRequestTimeout = time.Second * 5
	// DefaultPoolSize sets the connection pool size
	DefaultPoolSize = 0
	// DefaultPoolTTL sets the connection pool ttl
	DefaultPoolTTL = time.Minute
)

// Makes a synchronous call to a service using the default client
func Call(ctx context.Context, request Request, response interface{}, opts ...CallOption) error {
	return DefaultClient.Call(ctx, request, response, opts...)
}

// Makes a synchronous call to the specified address using the default client
func CallRemote(ctx context.Context, address string, request Request, response interface{}, opts ...CallOption) error {
	return DefaultClient.CallRemote(ctx, address, request, response, opts...)
}

// Creates a streaming connection with a service and returns responses on the
// channel passed in. It's up to the user to close the streamer.
func Stream(ctx context.Context, request Request, opts ...CallOption) (Streamer, error) {
	return DefaultClient.Stream(ctx, request, opts...)
}

// Creates a streaming connection to the address specified.
func StreamRemote(ctx context.Context, address string, request Request, opts ...CallOption) (Streamer, error) {
	return DefaultClient.StreamRemote(ctx, address, request, opts...)
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
func NewRequest(service, method string, request interface{}, reqOpts ...RequestOption) Request {
	return DefaultClient.NewRequest(service, method, request, reqOpts...)
}

// Creates a new protobuf request using the default client
func NewProtoRequest(service, method string, request interface{}, reqOpts ...RequestOption) Request {
	return DefaultClient.NewProtoRequest(service, method, request, reqOpts...)
}

// Creates a new json request using the default client
func NewJsonRequest(service, method string, request interface{}, reqOpts ...RequestOption) Request {
	return DefaultClient.NewJsonRequest(service, method, request, reqOpts...)
}

func String() string {
	return DefaultClient.String()
}
