// Package lock provides distributed locking
package lock

import (
	"time"
)

// Lock is a distributed locking interface
type Lock interface {
	// Acquire a lock with given id
	Acquire(id string, opts ...AcquireOption) error
	// Release the lock with given id
	Release(id string) error
}

type Options struct {
	Nodes  []string
	Prefix string
}

type AcquireOptions struct {
	TTL  time.Duration
	Wait time.Duration
}

type Option func(o *Options)
type AcquireOption func(o *AcquireOptions)
