package pool

import (
	"time"

	"github.com/asim/go-micro/v3/network"
)

type Options struct {
	Network network.Network
	TTL       time.Duration
	Size      int
}

type Option func(*Options)

func Size(i int) Option {
	return func(o *Options) {
		o.Size = i
	}
}

func Network(t network.Network) Option {
	return func(o *Options) {
		o.Network = t
	}
}

func TTL(t time.Duration) Option {
	return func(o *Options) {
		o.TTL = t
	}
}
