package transport

import (
	"time"

	"golang.org/x/net/context"
)

type Message struct {
	Header map[string]string
	Body   []byte
}

type Socket interface {
	Recv(*Message) error
	Send(*Message) error
	Close() error
}

type Client interface {
	Recv(*Message) error
	Send(*Message) error
	Close() error
}

type Listener interface {
	Addr() string
	Close() error
	Accept(func(Socket)) error
}

type Transport interface {
	Dial(addr string, opts ...DialOption) (Client, error)
	Listen(addr string) (Listener, error)
	String() string
}

type Options struct {
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type DialOptions struct {
	Stream  bool
	Timeout time.Duration

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type Option func(*Options)

type DialOption func(*DialOptions)

var (
	DefaultTransport Transport = newHttpTransport([]string{})

	DefaultDialTimeout = time.Second * 5
)

func WithStream() DialOption {
	return func(o *DialOptions) {
		o.Stream = true
	}
}

func WithTimeout(d time.Duration) DialOption {
	return func(o *DialOptions) {
		o.Timeout = d
	}
}

func NewTransport(addrs []string, opt ...Option) Transport {
	return newHttpTransport(addrs, opt...)
}

func Dial(addr string, opts ...DialOption) (Client, error) {
	return DefaultTransport.Dial(addr, opts...)
}

func Listen(addr string) (Listener, error) {
	return DefaultTransport.Listen(addr)
}

func String() string {
	return DefaultTransport.String()
}
