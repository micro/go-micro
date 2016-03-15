package http

import (
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/cmd"
)

func init() {
	cmd.DefaultBrokers["http"] = NewBroker
}

func NewBroker(opts ...broker.Option) broker.Broker {
	return broker.NewBroker(opts...)
}
