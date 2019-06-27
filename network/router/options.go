package router

import (
	"github.com/google/uuid"
	"github.com/micro/go-micro/registry"
)

var (
	// DefaultAddress is default router address
	DefaultAddress = ":9093"
)

// Options are router options
type Options struct {
	// ID is router id
	ID string
	// Address is router address
	Address string
	// Network is micro network
	Network string
	// Registry is the local registry
	Registry registry.Registry
	// Table is routing table
	Table Table
}

// ID sets Router ID
func ID(id string) Option {
	return func(o *Options) {
		o.ID = id
	}
}

// Address sets router service address
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// Network sets router network
func Network(n string) Option {
	return func(o *Options) {
		o.Network = n
	}
}

// RoutingTable sets the routing table
func RoutingTable(t Table) Option {
	return func(o *Options) {
		o.Table = t
	}
}

// Registry sets the local registry
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// DefaultOptions returns router default options
func DefaultOptions() Options {
	// NOTE: by default both local and network registies use default registry i.e. mdns
	return Options{
		ID:       uuid.New().String(),
		Address:  DefaultAddress,
		Registry: registry.DefaultRegistry,
		Table:    NewTable(),
		Network:  "local",
	}
}
