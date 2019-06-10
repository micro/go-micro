package router

import (
	"context"
)

// Options allows to set Router options
type Options struct {
	// ID is router ID
	ID string
	// Address is router address
	Address string
	// GossipAddr is router gossip address
	GossipAddr string
	// NetworkAddr defines micro network address
	NetworkAddr string
	// RIB is Routing Information Base
	RIB RIB
	// Table is routing table
	Table Table
	// Context stores arbitrary options
	Context context.Context
}

// ID sets Router ID
func ID(id string) Option {
	return func(o *Options) {
		o.ID = id
	}
}

// Address allows to set router address
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// GossipAddress allows to set router address
func GossipAddress(a string) Option {
	return func(o *Options) {
		o.GossipAddr = a
	}
}

// NetworkAddr allows to set router network
func NetworkAddr(n string) Option {
	return func(o *Options) {
		o.NetworkAddr = n
	}
}

// RIBase allows to configure RIB
func RIBase(r RIB) Option {
	return func(o *Options) {
		o.RIB = r
	}
}

// RoutingTable allows to specify custom routing table
func RoutingTable(t Table) Option {
	return func(o *Options) {
		o.Table = t
	}
}
