package tunnel

import (
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/transport/quic"
)

var (
	// DefaultAddress is default tunnel bind address
	DefaultAddress = ":0"
	// The shared default token
	DefaultToken = "micro"
)

type Option func(*Options)

// Options provides network configuration options
type Options struct {
	// Id is tunnel id
	Id string
	// Address is tunnel address
	Address string
	// Nodes are remote nodes
	Nodes []string
	// The shared auth token
	Token string
	// Transport listens to incoming connections
	Transport transport.Transport
}

type DialOption func(*DialOptions)

type DialOptions struct {
	// Link specifies the link to use
	Link string
	// specify a multicast connection
	Multicast bool
	// the dial timeout
	Timeout time.Duration
}

// The tunnel id
func Id(id string) Option {
	return func(o *Options) {
		o.Id = id
	}
}

// The tunnel address
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// Nodes specify remote network nodes
func Nodes(n ...string) Option {
	return func(o *Options) {
		o.Nodes = n
	}
}

// Token sets the shared token for auth
func Token(t string) Option {
	return func(o *Options) {
		o.Token = t
	}
}

// Transport listens for incoming connections
func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

// DefaultOptions returns router default options
func DefaultOptions() Options {
	return Options{
		Id:        uuid.New().String(),
		Address:   DefaultAddress,
		Token:     DefaultToken,
		Transport: quic.NewTransport(),
	}
}

// Dial options

// Dial multicast sets the multicast option to send only to those mapped
func DialMulticast() DialOption {
	return func(o *DialOptions) {
		o.Multicast = true
	}
}

// DialTimeout sets the dial timeout of the connection
func DialTimeout(t time.Duration) DialOption {
	return func(o *DialOptions) {
		o.Timeout = t
	}
}

// DialLink specifies the link to pin this connection to.
// This is not applicable if the multicast option is set.
func DialLink(id string) DialOption {
	return func(o *DialOptions) {
		o.Link = id
	}
}
