package transport

import (
	"time"
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
	Listen(addr string, opts ...ListenOption) (Listener, error)
	String() string
}

type Option func(*Options)

type DialOption func(*DialOptions)

type ListenOption func(*ListenOptions)

var (
	DefaultTransport Transport = newHttpTransport([]string{})

	DefaultDialTimeout = time.Second * 5
)

func NewTransport(addrs []string, opt ...Option) Transport {
	return newHttpTransport(addrs, opt...)
}

func Dial(addr string, opts ...DialOption) (Client, error) {
	return DefaultTransport.Dial(addr, opts...)
}

func Listen(addr string, opts ...ListenOption) (Listener, error) {
	return DefaultTransport.Listen(addr, opts...)
}

func String() string {
	return DefaultTransport.String()
}
