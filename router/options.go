package router

import (
	"context"
	"time"

	"github.com/micro/go-micro/registry"
)

// Options allows to set Router options
type Options struct {
	// Registry is route source registry i.e. local registry
	Registry registry.Registry
	// Context stores arbitrary options
	Context context.Context
}

// RouteOption allows to soecify routing table options
type RouteOption struct {
	// TTL defines route entry lifetime
	TTL time.Duration
	// COntext allows to specify other arbitrary options
	Context context.Context
}

// Registry is local registry
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}
