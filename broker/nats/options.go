package nats

import (
	"time"

	natsp "github.com/nats-io/nats.go"
	"go-micro.dev/v5/broker"
)

type optionsKey struct{}
type drainConnectionKey struct{}
type poolSizeKey struct{}
type poolIdleTimeoutKey struct{}

// Options accepts nats.Options.
func Options(opts natsp.Options) broker.Option {
	return setBrokerOption(optionsKey{}, opts)
}

// DrainConnection will drain subscription on close.
func DrainConnection() broker.Option {
	return setBrokerOption(drainConnectionKey{}, struct{}{})
}

// PoolSize sets the size of the connection pool.
// If set to a value > 1, the broker will use a connection pool.
// Default is 1 (no pooling).
func PoolSize(size int) broker.Option {
	return setBrokerOption(poolSizeKey{}, size)
}

// PoolIdleTimeout sets the timeout for idle connections in the pool.
// Connections idle for longer than this duration will be closed.
// Default is 5 minutes. Set to 0 to disable idle timeout.
func PoolIdleTimeout(timeout time.Duration) broker.Option {
	return setBrokerOption(poolIdleTimeoutKey{}, timeout)
}
