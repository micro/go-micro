package http

// This is a hack

import (
	"github.com/piemapping/go-micro/transport"
)

func NewTransport(addrs []string, opt ...transport.Option) transport.Transport {
	return transport.NewTransport(addrs, opt...)
}
