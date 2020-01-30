package nats

import (
	"github.com/micro/go-micro/v2/broker"
	nats "github.com/nats-io/nats.go"
)

type optionsKey struct{}
type drainConnectionKey struct{}
type localServerKey struct{}

// Options accepts nats.Options
func Options(opts nats.Options) broker.Option {
	return setBrokerOption(optionsKey{}, opts)
}

// LocalServer embeds a local server rather than connecting to one
func LocalServer() broker.Option {
	return setBrokerOption(localServerKey{}, true)
}

// DrainConnection will drain subscription on close
func DrainConnection() broker.Option {
	return setBrokerOption(drainConnectionKey{}, struct{}{})
}
