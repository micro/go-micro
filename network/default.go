package network

import (
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"
)

// network implements Network interface
type network struct {
	// options configure the network
	options Options
}

// newNetwork returns a new network node
func newNetwork(opts ...Option) Network {
	options := DefaultOptions()

	for _, o := range opts {
		o(&options)
	}

	return &network{
		options: options,
	}
}

// Name returns network name
func (n *network) Name() string {
	return n.options.Name
}

// Connect connects the network
func (n *network) Connect() error {
	return nil
}

// Close closes network connection
func (n *network) Close() error {
	return nil
}

// Client returns network client
func (n *network) Client() client.Client {
	return nil
}

// Server returns network server
func (n *network) Server() server.Server {
	return nil
}
