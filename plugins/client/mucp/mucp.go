// Package mucp provides an mucp client
package mucp

import (
	"go-micro.dev/v4/cmd"
	"go-micro.dev/v4/client"
)

func init() {
	cmd.DefaultClients["mucp"] = NewClient
}

// NewClient returns a new micro client interface
func NewClient(opts ...client.Option) client.Client {
	return client.NewClient(opts...)
}
