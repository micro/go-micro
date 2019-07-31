package network

import (
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"
)

type network struct {
	name    string
	options Options
}

func (n *network) Name() string {
	return n.name
}

func (n *network) Connect() error {
	return n.options.Server.Start()
}

func (n *network) Close() error {
	return n.options.Server.Stop()
}

// NewNetwork returns a new network node
func NewNetwork(opts ...Option) Network {
	options := Options{
		Name:   DefaultName,
		Client: client.DefaultClient,
		Server: server.DefaultServer,
	}

	for _, o := range opts {
		o(&options)
	}

	// set the server name
	options.Server.Init(
		server.Name(options.Name),
	)

	return &network{
		options: options,
	}
}
