package gossip

import (
	"context"

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
