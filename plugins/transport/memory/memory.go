// Package memory is an in-memory transport
package memory

import (
	"github.com/micro/go-micro/v2/cmd"
	"github.com/micro/go-micro/v2/transport"
	"github.com/micro/go-micro/v2/transport/memory"
)

func init() {
	cmd.DefaultTransports["memory"] = NewTransport
}

func NewTransport(opts ...transport.Option) transport.Transport {
	return memory.NewTransport(opts...)
}
