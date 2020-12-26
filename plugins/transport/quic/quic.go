// Package quic provides a QUIC based transport
package quic

import (
	"github.com/micro/go-micro/v2/cmd"
	"github.com/micro/go-micro/v2/transport"
	"github.com/micro/go-micro/v2/transport/quic"
)

func init() {
	cmd.DefaultTransports["quic"] = NewTransport
}

func NewTransport(opts ...transport.Option) transport.Transport {
	return quic.NewTransport(opts...)
}
