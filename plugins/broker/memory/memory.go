// Package memory provides a memory broker
package memory

import (
	"github.com/micro/go-micro/v2/broker"
	"github.com/micro/go-micro/v2/broker/memory"
	"github.com/micro/go-micro/v2/cmd"
)

func init() {
	cmd.DefaultBrokers["memory"] = NewBroker
}

func NewBroker(opts ...broker.Option) broker.Broker {
	return memory.NewBroker(opts...)
}
