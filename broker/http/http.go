package http

import (
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/cmd"
)

func init() {
	cmd.DefaultBrokers["http"] = NewBroker
}

func NewBroker(addrs []string, opts ...broker.Option) broker.Broker {
	return broker.NewBroker(addrs, opts...)
}
