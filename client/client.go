// Package client is an interface for an RPC client
package client

import (
	"context"
	"time"

	"github.com/micro/go-micro/v2/codec"
)

// Client is the interface used to make requests to services.
// It supports Request/Response via Transport and Publishing via the Broker.
// It also supports bidirectional streaming of requests.
type Client interface {
	Init(...Option) error
	Options() Options
	NewMessage(topic string, msg interface{}, opts ...MessageOption) Message
	NewRequest(service, endpoint string, req interface{}, reqOpts ...RequestOption) Request
	Call(ctx context.Context, req Request, rsp interface{}, opts ...CallOption) error
	Stream(ctx context.Context, req Request, opts ...CallOption) (Stream, error)
	Publish(ctx context.Context, msg Message, opts ...PublishOption) error
	String() string
}

// Router manages request routing
type Router interface {
	SendRequest(context.Context, Request) (Response, error)
}

// Message is the interface for publishing asynchronously
type Message interface {
	Topic() string
	Payload() interface{}
	ContentType() string
}

// Request is the interface for a synchronous request used by Call or Stream
type Request interface {
	// The service to call
	Service() string
	// The action to take
	Method() string
	// The endpoint to invoke
	Endpoint() string
	// The content type
	ContentType() string
	// The unencoded request body
	Body() interface{}
	// Write to the encoded request writer. This is nil before a call is made
	Codec() codec.Writer
	// indicates whether the request will be a streaming one rather than unary
	Stream() bool
}

// Response is the response received from a service
type Response interface {
	// Read the response
	Codec() codec.Reader
	// read the header
	Header() map[string]string
	// Read the undecoded response
	Read() ([]byte, error)
}

// Stream is the inteface for a bidirectional synchronous stream
type Stream interface {
	// Context for the stream
	Context() context.Context
	// The request made
	Request() Request
	// The response read
	Response() Response
	// Send will encode and send a request
	Send(interface{}) error
	// Recv will decode and read a response
	Recv(interface{}) error
	// Error returns the stream error
	Error() error
	// Close closes the stream
	Close() error
}

// Option used by the Client
type Option func(*Options)

// CallOption used by Call or Stream
type CallOption func(*CallOptions)

// PublishOption used by Publish
type PublishOption func(*PublishOptions)

// MessageOption used by NewMessage
type MessageOption func(*MessageOptions)

// RequestOption used by NewRequest
type RequestOption func(*RequestOptions)

var (
	// DefaultClient is a default client to use out of the box
	DefaultClient Client = newRpcClient()
	// DefaultBackoff is the default backoff function for retries
	DefaultBackoff = exponentialBackoff
	// DefaultRetry is the default check-for-retry function for retries
	DefaultRetry = RetryOnError
	// DefaultRetries is the default number of times a request is tried
	DefaultRetries = 1
	// DefaultRequestTimeout is the default request timeout
	DefaultRequestTimeout = time.Second * 5
	// DefaultPoolSize sets the connection pool size
	DefaultPoolSize = 100
	// DefaultPoolTTL sets the connection pool ttl
	DefaultPoolTTL = time.Minute

	// NewClient returns a new client
	NewClient func(...Option) Client = newRpcClient
)

// Makes a synchronous call to a service using the default client
func Call(ctx context.Context, request Request, response interface{}, opts ...CallOption) error {
	return DefaultClient.Call(ctx, request, response, opts...)
}

// Publishes a publication using the default client. Using the underlying broker
// set within the options.
func Publish(ctx context.Context, msg Message, opts ...PublishOption) error {
	return DefaultClient.Publish(ctx, msg, opts...)
}

// Creates a new message using the default client
func NewMessage(topic string, payload interface{}, opts ...MessageOption) Message {
	return DefaultClient.NewMessage(topic, payload, opts...)
}

// Creates a new request using the default client. Content Type will
// be set to the default within options and use the appropriate codec
func NewRequest(service, endpoint string, request interface{}, reqOpts ...RequestOption) Request {
	return DefaultClient.NewRequest(service, endpoint, request, reqOpts...)
}

// Creates a streaming connection with a service and returns responses on the
// channel passed in. It's up to the user to close the streamer.
func NewStream(ctx context.Context, request Request, opts ...CallOption) (Stream, error) {
	return DefaultClient.Stream(ctx, request, opts...)
}

func String() string {
	return DefaultClient.String()
}
