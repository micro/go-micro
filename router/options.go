package router

import (
	"context"

	"github.com/micro/go-micro/registry"
)

// Options allows to set Router options
type Options struct {
	// Registry is route source registry i.e. local registry
	Registry registry.Registry
	// Context stores arbitrary options
	Context context.Context
}

// Registry allows to set local service registry
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// RouteOptions allows to specify routing table options
type RouteOptions struct {
	// NetID is network ID
	NetID string
	// Metric is route metric
	Metric int
	// COntext allows to specify other arbitrary options
	Context context.Context
}

// NetID allows to set micro network ID
func NetID(id string) RouteOption {
	return func(o *RouteOptions) {
		o.NetID = id
	}
}

// Metric allows to set route cost metric
func Metric(m int) RouteOption {
	return func(o *RouteOptions) {
		o.Metric = m
	}
}
