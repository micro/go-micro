/*
Server represents a server instance in go-micro which handles synchronous
requests via handlers and asynchronous requests via subscribers that
register with a broker.

The server combines the all the packages in go-micro to create a whole unit
used for building applications including discovery, client/server communication
and pub/sub.

	import "github.com/micro/go-micro/server"

	type Greeter struct {}

	func (g *Greeter) Hello(ctx context.Context, req *greeter.Request, rsp *greeter.Response) error {
		rsp.Msg = "Hello " + req.Name
		return nil
	}

	s := server.NewServer()


	s.Handle(
		s.NewHandler(&Greeter{}),
	)

	s.Start()

*/
package server

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pborman/uuid"
	"golang.org/x/net/context"
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
	// indicates whether the request will be streamed
	Stream() bool
}

// Streamer represents a stream established with a client.
// A stream can be bidirectional which is indicated by the request.
// The last error will be left in Error().
// EOF indicated end of the stream.
type Streamer interface {
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

// Returns config options for the default service
func DefaultOptions() Options {
	return DefaultServer.Options()
}

// Initialises the default server with options passed in
func Init(opt ...Option) {
	if DefaultServer == nil {
		DefaultServer = newRpcServer(opt...)
	}
	DefaultServer.Init(opt...)
}

// Returns a new server with options passed in
func NewServer(opt ...Option) Server {
	return newRpcServer(opt...)
}

// Creates a new subscriber interface with the given topic
// and handler using the default server
func NewSubscriber(topic string, h interface{}, opts ...SubscriberOption) Subscriber {
	return DefaultServer.NewSubscriber(topic, h, opts...)
}

// Creates a new handler interface using the default server
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

// Registers a handler interface with the default server to
// handle inbound requests
func Handle(h Handler) error {
	return DefaultServer.Handle(h)
}

// Registers a subscriber interface with the default server
// which subscribes to specified topic with the broker
func Subscribe(s Subscriber) error {
	return DefaultServer.Subscribe(s)
}

// Registers the default server with the discovery system
func Register() error {
	return DefaultServer.Register()
}

// Deregisters the default server from the discovery system
func Deregister() error {
	return DefaultServer.Deregister()
}

// Blocking run starts the default server and waits for a kill
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
	log.Printf("Received signal %s", <-ch)

	if err := DefaultServer.Deregister(); err != nil {
		return err
	}

	return Stop()
}

// Starts the default server
func Start() error {
	config := DefaultServer.Options()
	log.Printf("Starting server %s id %s", config.Name, config.Id)
	return DefaultServer.Start()
}

// Stops the default server
func Stop() error {
	log.Printf("Stopping server")
	return DefaultServer.Stop()
}

func String() string {
	return DefaultServer.String()
}
