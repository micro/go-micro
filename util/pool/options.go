package pool

import (
	"time"

	"github.com/asim/nitro/app/network"
)

type Options struct {
	Transport network.Transport
	TTL       time.Duration
	Size      int
}

type Option func(*Options)

func Size(i int) Option {
	return func(o *Options) {
		o.Size = i
	}
}

func Transport(t network.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

func TTL(t time.Duration) Option {
	return func(o *Options) {
		o.TTL = t
	}
}
