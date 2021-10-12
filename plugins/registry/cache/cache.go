// Package cache provides a registry cache
package cache

import (
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/registry/cache"
)

// New returns a new cache
func New(r registry.Registry, opts ...cache.Option) cache.Cache {
	return cache.New(r, opts...)
}
