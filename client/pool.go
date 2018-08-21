package client

import (
	"context"
	"time"

	"github.com/micro/go-micro/transport"
)

// Pool is a connection pool.
type Pool interface {
	Conn(ctx context.Context, addr string) Conn
	Close() error
	Idle() int
	Options() PoolOptions
	Release(ctx context.Context, addr string, conn Conn, err error)
}

// Conn is a item in the client pool
type Conn interface {
	Created() time.Time
	transport.Client
}

// PoolOptions are options for the client pool.
type PoolOptions struct {
	Size int
	TTL  time.Duration
}

// PoolOption is a option for the client pool
type PoolOption func(*PoolOptions)

// WithSize sets the connection pool size
func WithSize(d int) PoolOption {
	return func(o *PoolOptions) {
		o.Size = d
	}
}

// WithTTL sets the connection pool size
func WithTTL(d time.Duration) PoolOption {
	return func(o *PoolOptions) {
		o.TTL = d
	}
}
