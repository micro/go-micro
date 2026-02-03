package nats

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"go-micro.dev/v5/transport"
)

type optionsKey struct{}
type poolSizeKey struct{}
type poolIdleTimeoutKey struct{}

// Options allow to inject a nats.Options struct for configuring
// the nats connection.
func Options(nopts nats.Options) transport.Option {
	return func(o *transport.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, optionsKey{}, nopts)
	}
}

// PoolSize sets the size of the connection pool.
// If set to a value > 1, the transport will use a connection pool.
// Default is 1 (no pooling).
func PoolSize(size int) transport.Option {
	return func(o *transport.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, poolSizeKey{}, size)
	}
}

// PoolIdleTimeout sets the timeout for idle connections in the pool.
// Connections idle for longer than this duration will be closed.
// Default is 5 minutes. Set to 0 to disable idle timeout.
func PoolIdleTimeout(timeout time.Duration) transport.Option {
	return func(o *transport.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, poolIdleTimeoutKey{}, timeout)
	}
}
