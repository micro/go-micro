package gossip

import (
	"context"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/micro/go-micro/registry"
)

type contextSecretKey struct{}

// Secret specifies an encryption key. The value should be either
// 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
func Secret(k []byte) registry.Option {
	return setRegistryOption(contextSecretKey{}, k)
}

type contextAddress struct{}

// Address to bind to - host:port
func Address(a string) registry.Option {
	return setRegistryOption(contextAddress{}, a)
}

type contextConfig struct{}

// Config allow to inject a *memberlist.Config struct for configuring gossip
func Config(c *memberlist.Config) registry.Option {
	return setRegistryOption(contextConfig{}, c)
}

type contextAdvertise struct{}

// The address to advertise for other gossip members - host:port
func Advertise(a string) registry.Option {
	return setRegistryOption(contextAdvertise{}, a)
}

type contextContext struct{}

// Context specifies a context for the registry.
// Can be used to signal shutdown of the registry.
// Can be used for extra option values.
func Context(ctx context.Context) registry.Option {
	return setRegistryOption(contextContext{}, ctx)
}

type connectTimeout struct{}

// ConnectTimeout specify registry connect timeout use -1 to specify infinite
func ConnectTimeout(td time.Duration) registry.Option {
	return setRegistryOption(connectTimeout{}, td)
}

type connectRetry struct{}

// ConnectRetry enable reconnect to registry then connection closed,
// use with ConnectTimeout to specify how long retry
func ConnectRetry(v bool) registry.Option {
	return setRegistryOption(connectRetry{}, v)
}
