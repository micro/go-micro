package gossip

import (
	"context"

	"github.com/hashicorp/memberlist"
	"github.com/micro/go-micro/registry"
)

type contextSecretKey struct{}

// Secret specifies an encryption key. The value should be either
// 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func Secret(k []byte) registry.Option {
	return func(o *registry.Options) {
		o.Context = context.WithValue(o.Context, contextSecretKey{}, k)
	}
}

type contextAddress struct{}

// Address to bind to - host:port
func Address(a string) registry.Option {
	return func(o *registry.Options) {
		o.Context = context.WithValue(o.Context, contextAddress{}, a)
	}
}

type contextConfig struct{}

// Config allow to inject a *memberlist.Config struct for configuring gossip
func Config(c *memberlist.Config) registry.Option {
	return func(o *registry.Options) {
		o.Context = context.WithValue(o.Context, contextConfig{}, c)
	}
}

type contextAdvertise struct{}

// The address to advertise for other gossip members - host:port
func Advertise(a string) registry.Option {
	return func(o *registry.Options) {
		o.Context = context.WithValue(o.Context, contextAdvertise{}, a)
	}
}
