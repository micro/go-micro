// Package server is an interface for a micro server
package server

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/micro/go-log"
	"github.com/pborman/uuid"
)

type Server interface {
	Options() Options
	Init(...Option) error
	Handle(Handler) error
	NewHandler(interface{}, ...HandlerOption) Handler
	NewSubscriber(string, interface{}, ...SubscriberOption) Subscriber
	Subscribe(Subscriber) error
	Register() error
	Deregister() error
	Start() error
	Stop() error
	String() string
}

type Message interface {
	Topic() string
	Payload() interface{}
	ContentType() string
}

type Request interface {
	Service() string
	Method() string
	ContentType() string
	Request() interface{}
	// indicates whether the request will be streamed
	Stream() bool
}

// Stream represents a stream established with a client.
// A stream can be bidirectional which is indicated by the request.
// The last error will be left in Error().
// EOF indicated end of the stream.
type Stream interface {
	Context() context.Context
	Request() Request
	Send(interface{}) error
	Recv(interface{}) error
	Error() error
	Close() error
}

type Option func(*Options)

type HandlerOption func(*HandlerOptions)

type SubscriberOption func(*SubscriberOptions)

var (
	DefaultAddress        = ":0"
	DefaultName           = "go-server"
	DefaultVersion        = "1.0.0"
	DefaultId             = uuid.NewUUID().String()
	DefaultServer  Server = newRpcServer()
)

// DefaultOptions returns config options for the default service
func DefaultOptions() Options {
	return DefaultServer.Options()
}

// Init initialises the default server with options passed in
func Init(opt ...Option) {
	if DefaultServer == nil {
		DefaultServer = newRpcServer(opt...)
	}
	DefaultServer.Init(opt...)
}

// NewServer returns a new server with options passed in
func NewServer(opt ...Option) Server {
	return newRpcServer(opt...)
}

// NewSubscriber creates a new subscriber interface with the given topic
// and handler using the default server
func NewSubscriber(topic string, h interface{}, opts ...SubscriberOption) Subscriber {
	return DefaultServer.NewSubscriber(topic, h, opts...)
}

// NewHandler creates a new handler interface using the default server
// Handlers are required to be a public object with public
// methods. Call to a service method such as Foo.Bar expects
// the type:
//
//	type Foo struct {}
//	func (f *Foo) Bar(ctx, req, rsp) error {
//		return nil
//	}
//
func NewHandler(h interface{}, opts ...HandlerOption) Handler {
	return DefaultServer.NewHandler(h, opts...)
}

// Handle registers a handler interface with the default server to
// handle inbound requests
func Handle(h Handler) error {
	return DefaultServer.Handle(h)
}

// Subscribe registers a subscriber interface with the default server
// which subscribes to specified topic with the broker
func Subscribe(s Subscriber) error {
	return DefaultServer.Subscribe(s)
}

// Register registers the default server with the discovery system
func Register() error {
	return DefaultServer.Register()
}

// Deregister deregisters the default server from the discovery system
func Deregister() error {
	return DefaultServer.Deregister()
}

// Run starts the default server and waits for a kill
// signal before exiting. Also registers/deregisters the server
func Run() error {
	if err := Start(); err != nil {
		return err
	}

	if err := DefaultServer.Register(); err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	log.Logf("Received signal %s", <-ch)

	if err := DefaultServer.Deregister(); err != nil {
		return err
	}

	return Stop()
}

// Start starts the default server
func Start() error {
	config := DefaultServer.Options()
	log.Logf("Starting server %s id %s", config.Name, config.Id)
	return DefaultServer.Start()
}

// Stop stops the default server
func Stop() error {
	log.Logf("Stopping server")
	return DefaultServer.Stop()
}

// String returns name of Server implementation
func String() string {
	return DefaultServer.String()
}
