// Package mucp provides an mucp client
package mucp

import (
	"github.com/chinahtl/go-micro/v3/cmd"
	"github.com/chinahtl/go-micro/v3/client"
)

func init() {
	cmd.DefaultClients["mucp"] = NewClient
}

// NewClient returns a new micro client interface
func NewClient(opts ...client.Option) client.Client {
	return client.NewClient(opts...)
}
