// Package server is an interface for a micro server
package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v3/codec"
	"github.com/micro/go-micro/v3/registry"
)

// Server is a simple micro server abstraction
type Server interface {
	// Initialise options
	Init(...Option) error
	// Retrieve the options
	Options() Options
	// Register a handler
	Handle(Handler) error
	// Create a new handler
	NewHandler(interface{}, ...HandlerOption) Handler
	// Create a new subscriber
	NewSubscriber(string, interface{}, ...SubscriberOption) Subscriber
	// Register a subscriber
	Subscribe(Subscriber) error
	// Start the server
	Start() error
	// Stop the server
	Stop() error
	// Server implementation
	String() string
}

// Router handle serving messages
type Router interface {
	// ProcessMessage processes a message
	ProcessMessage(context.Context, Message) error
	// ServeRequest processes a request to completion
	ServeRequest(context.Context, Request, Response) error
}

// Message is an async message interface
type Message interface {
	// Topic of the message
	Topic() string
	// The decoded payload value
	Payload() interface{}
	// The content type of the payload
	ContentType() string
	// The raw headers of the message
	Header() map[string]string
	// The raw body of the message
	Body() []byte
	// Codec used to decode the message
	Codec() codec.Reader
}

// Request is a synchronous request interface
type Request interface {
	// Service name requested
	Service() string
	// The action requested
	Method() string
	// Endpoint name requested
	Endpoint() string
	// Content type provided
	ContentType() string
	// Header of the request
	Header() map[string]string
	// Body is the initial decoded value
	Body() interface{}
	// Read the undecoded request body
	Read() ([]byte, error)
	// The encoded message stream
	Codec() codec.Reader
	// Indicates whether its a stream
	Stream() bool
}

// Response is the response writer for unencoded messages
type Response interface {
	// Encoded writer
	Codec() codec.Writer
	// Write the header
	WriteHeader(map[string]string)
	// write a response directly to the client
	Write([]byte) error
}

// Stream represents a stream established with a client.
// A stream can be bidirectional which is indicated by the request.
// The last error will be left in Error().
// EOF indicates end of the stream.
type Stream interface {
	Context() context.Context
	Request() Request
	Send(interface{}) error
	Recv(interface{}) error
	Error() error
	Close() error
}

// Handler interface represents a request handler. It's generated
// by passing any type of public concrete object with endpoints into server.NewHandler.
// Most will pass in a struct.
//
// Example:
//
//      type Greeter struct {}
//
//      func (g *Greeter) Hello(context, request, response) error {
//              return nil
//      }
//
type Handler interface {
	Name() string
	Handler() interface{}
	Endpoints() []*registry.Endpoint
	Options() HandlerOptions
}

// Subscriber interface represents a subscription to a given topic using
// a specific subscriber function or object with endpoints. It mirrors
// the handler in its behaviour.
type Subscriber interface {
	Topic() string
	Subscriber() interface{}
	Endpoints() []*registry.Endpoint
	Options() SubscriberOptions
}

type Option func(*Options)

var (
	DefaultAddress          = ":0"
	DefaultName             = "go.micro.server"
	DefaultVersion          = "latest"
	DefaultId               = uuid.New().String()
	DefaultRegisterCheck    = func(context.Context) error { return nil }
	DefaultRegisterInterval = time.Second * 30
	DefaultRegisterTTL      = time.Second * 90
)
