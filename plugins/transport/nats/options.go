package nats

import (
	"context"

	"go-micro.dev/v4/transport"
	"github.com/nats-io/nats.go"
)

type optionsKey struct{}

// Options allow to inject a nats.Options struct for configuring
// the nats connection
func Options(nopts nats.Options) transport.Option {
	return func(o *transport.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, optionsKey{}, nopts)
	}
}
