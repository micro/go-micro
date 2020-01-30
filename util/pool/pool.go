// Package pool is a connection pool
package pool

import (
	"time"

	"github.com/micro/go-micro/v2/transport"
)

// Pool is an interface for connection pooling
type Pool interface {
	// Close the pool
	Close() error
	// Get a connection
	Get(addr string, opts ...transport.DialOption) (Conn, error)
	// Releaes the connection
	Release(c Conn, status error) error
}

type Conn interface {
	// unique id of connection
	Id() string
	// time it was created
	Created() time.Time
	// embedded connection
	transport.Client
}

func NewPool(opts ...Option) Pool {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	return newPool(options)
}
