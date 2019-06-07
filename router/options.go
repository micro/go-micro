package router

import (
	"context"
)

// Options allows to set Router options
type Options struct {
	// Address is router address
	Address string
	// RIB is Routing Information Base
	RIB RIB
	// Table is routing table
	Table Table
	// Context stores arbitrary options
	Context context.Context
}

// RIBase allows to configure RIB
func RIBase(r RIB) Option {
	return func(o *Options) {
		o.RIB = r
	}
}

// Address allows to set router address
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// RoutingTable allows to specify custom routing table
func RoutingTable(t Table) Option {
	return func(o *Options) {
		o.Table = t
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
