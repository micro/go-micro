package nats

import (
	"github.com/micro/go-micro/broker"
	nats "github.com/nats-io/nats.go"
)

type optionsKey struct{}
type drainConnectionKey struct{}
type drainSubscriptionKey struct{}

// Options accepts nats.Options
func Options(opts nats.Options) broker.Option {
	return setBrokerOption(optionsKey{}, opts)
}

// DrainConnection will drain subscription on close
func DrainConnection() broker.Option {
	return setBrokerOption(drainConnectionKey{}, true)
}

// DrainSubscription will drain pending messages when unsubscribe
func DrainSubscription() broker.SubscribeOption {
	return setSubscribeOption(drainSubscriptionKey{}, true)
}
