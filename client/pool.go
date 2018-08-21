package client

import (
	"context"
	"time"

	"github.com/micro/go-micro/transport"
)

// Pool is a connection pool.
type Pool interface {
	Conn(ctx context.Context, addr string) Conn
	Release(ctx context.Context, addr string, conn Conn, err error)
	Idle() int
	Options() PoolOptions
	Close() error
}

// Conn is a item in the client pool
type Conn interface {
	Created() time.Time
	transport.Client
}

// PoolOptions are options for the client pool.
type PoolOptions struct {
	PoolSize int
	PoolTTL  time.Duration
}

// PoolOption is a option for the client pool
type PoolOption func(*PoolOptions)

// WithPoolSize sets the connection pool size
func WithPoolSize(d int) PoolOption {
	return func(o *PoolOptions) {
		o.PoolSize = d
	}
}

// TTL sets the connection pool size
func WithPoolTTL(d time.Duration) PoolOption {
	return func(o *PoolOptions) {
		o.PoolTTL = d
	}
}
