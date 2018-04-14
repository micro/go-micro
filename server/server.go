// Package server is an interface for a micro server
package server

import (
	"context"

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

// NewServer returns a new server with options passed in
func NewServer(opt ...Option) Server {
	return newRpcServer(opt...)
}
