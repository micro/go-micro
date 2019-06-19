package router

import (
	"github.com/google/uuid"
	"github.com/micro/go-micro/registry"
)

var (
	// DefaultAddress is default router address
	DefaultAddress = ":9093"
	// DefaultAdvertise is default address advertised to the network
	DefaultAdvertise = ":9094"
)

// Options are router options
type Options struct {
	// ID is router id
	ID string
	// Address is router address
	Address string
	// Advertise is the address advertised to the network
	Advertise string
	// Registry is the local registry
	Registry registry.Registry
	// Networkis the network registry
	Network registry.Registry
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

// Advertise sets the address that is advertise to the network
func Advertise(n string) Option {
	return func(o *Options) {
		o.Advertise = n
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

// Network sets the network registry
func Network(r registry.Registry) Option {
	return func(o *Options) {
		o.Network = r
	}
}

// DefaultOptions returns router default options
func DefaultOptions() Options {
	// NOTE: by default both local and network registies use default registry i.e. mdns
	return Options{
		ID:        uuid.New().String(),
		Address:   DefaultAddress,
		Advertise: DefaultAdvertise,
		Registry:  registry.DefaultRegistry,
		Network:   registry.DefaultRegistry,
		Table:     NewTable(),
	}
}
