// Package rpc provides an rpc client
package rpc

import (
	"github.com/micro/go-micro/client"
)

// NewClient returns a new micro client interface
func NewClient(opts ...client.Option) client.Client {
	return client.NewClient(opts...)
}
