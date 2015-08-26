package http

// This is a hack

import (
	"github.com/kynrai/go-micro/broker"
)

func NewBroker(addrs []string, opt ...broker.Option) broker.Broker {
	return broker.NewBroker(addrs, opt...)
}
