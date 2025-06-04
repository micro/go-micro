// Package server is an interface for a micro server
package server

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"

	"go-micro.dev/v5/codec"
	log "go-micro.dev/v5/logger"
	"go-micro.dev/v5/registry"
	signalutil "go-micro.dev/v5/util/signal"
)

// Server is a simple micro server abstraction.
type Server interface {
	// Initialize options
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

// Router handle serving messages.
type Router interface {
	// ProcessMessage processes a message
	ProcessMessage(context.Context, string, Message) error
	// ServeRequest processes a request to completion
	ServeRequest(context.Context, Request, Response) error
}

// Message is an async message interface.
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

// Request is a synchronous request interface.
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

// Response is the response writer for unencoded messages.
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
//	type Greeter struct {}
//
//	func (g *Greeter) Hello(context, request, response) error {
//	        return nil
//	}
type Handler interface {
	Name() string
	Handler() interface{}
	Endpoints() []*registry.Endpoint
	Options() HandlerOptions
}

// Subscriber interface represents a subscription to a given topic using
// a specific subscriber function or object with endpoints. It mirrors
// the handler in its behavior.
type Subscriber interface {
	Topic() string
	Subscriber() interface{}
	Endpoints() []*registry.Endpoint
	Options() SubscriberOptions
}

type Option func(*Options)

var (
	DefaultAddress                 = ":0"
	DefaultName                    = "go.micro.server"
	DefaultVersion                 = "latest"
	DefaultId                      = uuid.New().String()
	DefaultServer           Server = NewRPCServer()
	DefaultRouter                  = newRpcRouter()
	DefaultRegisterCheck           = func(context.Context) error { return nil }
	DefaultRegisterInterval        = time.Second * 30
	DefaultRegisterTTL             = time.Second * 90

	// NewServer creates a new server.
	NewServer func(...Option) Server = NewRPCServer
)

// DefaultOptions returns config options for the default service.
func DefaultOptions() Options {
	return DefaultServer.Options()
}

func Init(opt ...Option) {
	if DefaultServer == nil {
		DefaultServer = NewRPCServer(opt...)
	}
	DefaultServer.Init(opt...)
}

// NewRouter returns a new router.
func NewRouter() *router {
	return newRpcRouter()
}

// NewSubscriber creates a new subscriber interface with the given topic
// and handler using the default server.
func NewSubscriber(topic string, h interface{}, opts ...SubscriberOption) Subscriber {
	return DefaultServer.NewSubscriber(topic, h, opts...)
}

// NewHandler creates a new handler interface using the default server
// Handlers are required to be a public object with public
// endpoints. Call to a service endpoint such as Foo.Bar expects
// the type:
//
//	type Foo struct {}
//	func (f *Foo) Bar(ctx, req, rsp) error {
//		return nil
//	}
func NewHandler(h interface{}, opts ...HandlerOption) Handler {
	return DefaultServer.NewHandler(h, opts...)
}

// Handle registers a handler interface with the default server to
// handle inbound requests.
func Handle(h Handler) error {
	return DefaultServer.Handle(h)
}

// Subscribe registers a subscriber interface with the default server
// which subscribes to specified topic with the broker.
func Subscribe(s Subscriber) error {
	return DefaultServer.Subscribe(s)
}

// Run starts the default server and waits for a kill
// signal before exiting. Also registers/deregisters the server.
func Run() error {
	if err := Start(); err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signalutil.Shutdown()...)
	DefaultServer.Options().Logger.Logf(log.InfoLevel, "Received signal %s", <-ch)

	return Stop()
}

// Start starts the default server.
func Start() error {
	config := DefaultServer.Options()
	config.Logger.Logf(log.InfoLevel, "Starting server %s id %s", config.Name, config.Id)
	return DefaultServer.Start()
}

// Stop stops the default server.
func Stop() error {
	DefaultServer.Options().Logger.Logf(log.InfoLevel, "Stopping server")
	return DefaultServer.Stop()
}

// String returns name of Server implementation.
func String() string {
	return DefaultServer.String()
}
