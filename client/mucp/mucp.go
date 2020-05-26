// Package mucp provides an mucp client
package mucp

import (
	"github.com/micro/go-micro/v2/client"
)

// NewClient returns a new micro client interface
func NewClient(opts ...client.Option) client.Client {
	return client.NewClient(opts...)
}
