package service

import (
	"context"

	"github.com/micro/go-micro/v2/broker"
	"github.com/micro/go-micro/v2/client"
)

type clientKey struct{}

// Client to call broker service
func Client(c client.Client) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.WithValue(context.Background(), clientKey{}, c)
			return
		}

		o.Context = context.WithValue(o.Context, clientKey{}, c)
	}
}
