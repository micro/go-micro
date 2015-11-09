package http

// This is a hack

import (
	"github.com/piemapping/go-micro/broker"
)

// NewBroker instantiates and returns a new HTTP-based broker
func NewBroker(addrs []string, opt ...broker.Option) broker.Broker {
	return broker.NewBroker(addrs, opt...)
}
