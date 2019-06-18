package router

import (
	"fmt"
	"strings"

	"github.com/olekukonko/tablewriter"
)

var (
	// DefaultLocalMetric is default route cost for local network
	DefaultLocalMetric = 1
	// DefaultNetworkMetric is default route cost for micro network
	DefaultNetworkMetric = 10
)

// AddPolicy defines routing table addition policy
type AddPolicy int

const (
	// OverrideIfExists overrides route if it already exists
	OverrideIfExists AddPolicy = iota
	// IgnoreIfExists does not modify existing route
	IgnoreIfExists
)

// String returns human reprensentation of policy
func (p AddPolicy) String() string {
	switch p {
	case OverrideIfExists:
		return "OVERRIDE"
	case IgnoreIfExists:
		return "IGNORE"
	default:
		return "UNKNOWN"
	}
}

// RouteOption is used to set routing table entry options
type RouteOption func(*RouteOptions)

// RouteOptions are route options
type RouteOptions struct {
	// DestAddr is destination address
	DestAddr string
	// Gateway is the next route hop
	Gateway Router
	// Network defines micro network
	Network string
	// Metric is route cost metric
	Metric int
	// Policy defines route addition policy
	Policy AddPolicy
}

// DestAddr sets destination address
func DestAddr(a string) RouteOption {
	return func(o *RouteOptions) {
		o.DestAddr = a
	}
}

// Gateway sets the route gateway
func Gateway(r Router) RouteOption {
	return func(o *RouteOptions) {
		o.Gateway = r
	}
}

// Network sets micro network
func Network(n string) RouteOption {
	return func(o *RouteOptions) {
		o.Network = n
	}
}

// Metric sets route metric
func Metric(m int) RouteOption {
	return func(o *RouteOptions) {
		o.Metric = m
	}
}

// RoutePolicy sets add route policy
func RoutePolicy(p AddPolicy) RouteOption {
	return func(o *RouteOptions) {
		o.Policy = p
	}
}

// Route is routing table route
type Route interface {
	// Options returns route options
	Options() RouteOptions
}

type route struct {
	opts RouteOptions
}

// NewRoute returns new routing table route
func NewRoute(opts ...RouteOption) Route {
	eopts := RouteOptions{}

	for _, o := range opts {
		o(&eopts)
	}

	return &route{
		opts: eopts,
	}
}

// Options returns route options
func (r *route) Options() RouteOptions {
	return r.opts
}

// String allows to print the route
func (r *route) String() string {
	// this will help us build routing table string
	sb := &strings.Builder{}

	// create nice table printing structure
	table := tablewriter.NewWriter(sb)
	table.SetHeader([]string{"Destination", "Gateway", "Network", "Metric"})

	strRoute := []string{
		r.opts.DestAddr,
		r.opts.Gateway.Address(),
		r.opts.Network,
		fmt.Sprintf("%d", r.opts.Metric),
	}
	table.Append(strRoute)

	// render table into sb
	table.Render()

	return sb.String()
}
