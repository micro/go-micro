package stomp

import (
	"context"

	"github.com/asim/go-micro/v3/broker"
)

// Context related keys and funcs
type authKey struct{}
type connectHeaderKey struct{}
type connectTimeoutKey struct{}
type durableQueueKey struct{}
type receiptKey struct{}
type subscribeHeaderKey struct{}
type suppressContentLengthKey struct{}
type vHostKey struct{}

type authRecord struct {
	username string
	password string
}

// setSubscribeOption returns a function to setup a context with given value
func setSubscribeOption(k, v interface{}) broker.SubscribeOption {
	return func(o *broker.SubscribeOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

// setBrokerOption returns a function to setup a context with given value
func setBrokerOption(k, v interface{}) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

// setPublishOption returns a function to setup a context with given value
func setPublishOption(k, v interface{}) broker.PublishOption {
	return func(o *broker.PublishOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}
