package pool

import (
	"time"

	"go-micro.dev/v4/transport"
)

type Options struct {
	Transport transport.Transport

	// Only valid for pool.
	TTL time.Duration
	// Use MaxIdleConns plz.
	Size int

	IdleConnTimeout time.Duration
	MaxIdleConns    int
	MaxIdleConnsPer int
	MaxConnsPer     int

	// Use limitPool replace pool.
	UseLimitPool bool
}

type Option func(*Options)

func Size(i int) Option {
	return MaxIdleConns(i)
}

func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

func TTL(t time.Duration) Option {
	return func(o *Options) {
		o.TTL = t
	}
}

func MaxIdleConns(n int) Option {
	return func(o *Options) {
		o.MaxIdleConns = n
		o.Size = n
	}
}

func IdleConnTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.IdleConnTimeout = d
	}
}

func MaxIdleConnsPer(n int) Option {
	return func(o *Options) {
		o.MaxIdleConnsPer = n
	}
}

func MaxConnsPer(n int) Option {
	return func(o *Options) {
		o.MaxConnsPer = n
	}
}

func UseLimitPool() Option {
	return func(o *Options) {
		o.UseLimitPool = true
	}
}
