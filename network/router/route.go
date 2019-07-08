package router

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/olekukonko/tablewriter"
)

var (
	// DefaultLocalMetric is default route cost metric for the local network
	DefaultLocalMetric = 1
	// DefaultNetworkMetric is default route cost metric for the micro network
	DefaultNetworkMetric = 10
)

// RoutePolicy defines routing table policy
type RoutePolicy int

const (
	// Insert inserts a new route if it does not already exist
	Insert RoutePolicy = iota
	// Override overrides the route if it already exists
	Override
	// Skip skips modifying the route if it already exists
	Skip
)

// String returns human reprensentation of policy
func (p RoutePolicy) String() string {
	switch p {
	case Insert:
		return "INSERT"
	case Override:
		return "OVERRIDE"
	case Skip:
		return "SKIP"
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
	// Router is the router address
	Router string
	// Network is network address
	Network string
	// Metric is the route cost metric
	Metric int
	// Policy defines route policy
	Policy RoutePolicy
}

// Hash returns route hash sum.
func (r *Route) Hash() uint64 {
	h := fnv.New64()
	h.Reset()
	h.Write([]byte(r.Destination + r.Gateway + r.Network))

	return h.Sum64()
}

// String returns human readable route
func (r Route) String() string {
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
