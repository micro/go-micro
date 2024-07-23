package pool

import (
	"time"

	"go-micro.dev/v5/transport"
)

type Options struct {
	Transport    transport.Transport
	TTL          time.Duration
	CloseTimeout time.Duration
	Size         int
}

type Option func(*Options)

func Size(i int) Option {
	return func(o *Options) {
		o.Size = i
	}
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

func CloseTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.CloseTimeout = t
	}
}
