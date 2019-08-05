package tunnel

import (
	"github.com/google/uuid"
	"github.com/micro/go-micro/transport"
)

var (
	// DefaultAddress is default tunnel bind address
	DefaultAddress = ":9096"
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
	// Transport listens to incoming connections
	Transport transport.Transport
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
func Nodes(n []string) Option {
	return func(o *Options) {
		o.Nodes = n
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
		Nodes:     make([]string, 0),
		Transport: transport.DefaultTransport,
	}
}
