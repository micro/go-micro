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

// RoutePolicy defines routing table addition policy
type RoutePolicy int

const (
	// AddIfNotExist adds the route if it does not exist
	AddIfNotExists RoutePolicy = iota
	// OverrideIfExists overrides route if it already exists
	OverrideIfExists
	// IgnoreIfExists instructs to not modify existing route
	IgnoreIfExists
)

// String returns human reprensentation of policy
func (p RoutePolicy) String() string {
	switch p {
	case AddIfNotExists:
		return "ADD_IF_NOT_EXISTS"
	case OverrideIfExists:
		return "OVERRIDE_IF_EXISTS"
	case IgnoreIfExists:
		return "IGNORE_IF_EXISTS"
	default:
		return "UNKNOWN"
	}
}

// Route is network route
type Route struct {
	// Destination is destination address
	Destination string
	// Gateway is route gateway
	Gateway string
	// Router is the network router address
	Router string
	// Network is micro network address
	Network string
	// Metric is the route cost metric
	Metric int
	// Policy defines route policy
	Policy RoutePolicy
}

// String allows to print the route
func (r *Route) String() string {
	// this will help us build routing table string
	sb := &strings.Builder{}

	// create nice table printing structure
	table := tablewriter.NewWriter(sb)
	table.SetHeader([]string{"Destination", "Gateway", "Router", "Network", "Metric"})

	strRoute := []string{
		r.Destination,
		r.Gateway,
		r.Router,
		r.Network,
		fmt.Sprintf("%d", r.Metric),
	}
	table.Append(strRoute)

	// render table into sb
	table.Render()

	return sb.String()
}
