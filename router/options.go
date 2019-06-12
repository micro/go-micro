package router

import (
	"github.com/google/uuid"
	"github.com/micro/go-micro/registry"
)

var (
	// DefaultAddress is default router bind address
	DefaultAddress = ":9093"
	// DefaultNetworkAddress is default micro network bind address
	DefaultNetworkAddress = ":9094"
)

// Options allows to set router options
type Options struct {
	// ID is router ID
	ID string
	// Address is router address
	Address string
	// GossipAddress is router gossip address
	GossipAddress string
	// NetworkAddress is micro network address
	NetworkAddress string
	// LocalRegistry is router local registry
	LocalRegistry registry.Registry
	// NetworkRegistry is router remote registry
	NetworkRegistry registry.Registry
	// Table is routing table
	Table Table
	// RIB is Routing Information Base
	RIB RIB
}

// ID sets Router ID
func ID(id string) Option {
	return func(o *Options) {
		o.ID = id
	}
}

// Address sets router address
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// GossipAddress sets router gossip address
func GossipAddress(a string) Option {
	return func(o *Options) {
		o.GossipAddress = a
	}
}

// NetworkAddress sets router network address
func NetworkAddress(n string) Option {
	return func(o *Options) {
		o.NetworkAddress = n
	}
}

// RoutingTable allows to specify custom routing table
func RoutingTable(t Table) Option {
	return func(o *Options) {
		o.Table = t
	}
}

// LocalRegistry allows to specify local registry
func LocalRegistry(r registry.Registry) Option {
	return func(o *Options) {
		o.LocalRegistry = r
	}
}

// NetworkRegistry allows to specify remote registry
func NetworkRegistry(r registry.Registry) Option {
	return func(o *Options) {
		o.NetworkRegistry = r
	}
}

// RouterIB allows to configure RIB
func RouterIB(r RIB) Option {
	return func(o *Options) {
		o.RIB = r
	}
}

// DefaultOptions returns router default options
func DefaultOptions() Options {
	// NOTE: by default both local and network registies use default registry i.e. mdns
	// TODO: DefaultRIB needs to be added once it's properly figured out
	return Options{
		ID:              uuid.New().String(),
		Address:         DefaultAddress,
		NetworkAddress:  DefaultNetworkAddress,
		LocalRegistry:   registry.DefaultRegistry,
		NetworkRegistry: registry.DefaultRegistry,
		Table:           NewTable(),
	}
}
