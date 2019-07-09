package table

import (
	"fmt"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// LookupPolicy defines query policy
type LookupPolicy int

const (
	// DiscardIfNone discards query when no route is found
	DiscardIfNone LookupPolicy = iota
	// ClosestMatch returns closest match to supplied query
	ClosestMatch
)

// String returns human representation of LookupPolicy
func (lp LookupPolicy) String() string {
	switch lp {
	case DiscardIfNone:
		return "DISCARD"
	case ClosestMatch:
		return "CLOSEST"
	default:
		return "UNKNOWN"
	}
}

// QueryOption sets routing table query options
type QueryOption func(*QueryOptions)

// QueryOptions are routing table query options
type QueryOptions struct {
	// Service is destination service name
	Service string
	// Gateway is route gateway
	Gateway string
	// Network is network address
	Network string
	// Policy is query lookup policy
	Policy LookupPolicy
}

// QueryService sets destination address
func QueryService(s string) QueryOption {
	return func(o *QueryOptions) {
		o.Service = s
	}
}

// QueryGateway sets route gateway
func QueryGateway(g string) QueryOption {
	return func(o *QueryOptions) {
		o.Gateway = g
	}
}

// QueryNetwork sets route network address
func QueryNetwork(n string) QueryOption {
	return func(o *QueryOptions) {
		o.Network = n
	}
}

// QueryPolicy sets query policy
// NOTE: this might be renamed to filter or some such
func QueryPolicy(p LookupPolicy) QueryOption {
	return func(o *QueryOptions) {
		o.Policy = p
	}
}

// Query is routing table query
type Query interface {
	// Options returns query options
	Options() QueryOptions
}

// query is a basic implementation of Query
type query struct {
	opts QueryOptions
}

// NewQuery creates new query and returns it
func NewQuery(opts ...QueryOption) Query {
	// default options
	// NOTE: by default we use DefaultNetworkMetric
	qopts := QueryOptions{
		Service: "*",
		Gateway: "*",
		Network: "*",
		Policy:  DiscardIfNone,
	}

	for _, o := range opts {
		o(&qopts)
	}

	return &query{
		opts: qopts,
	}
}

// Options returns query options
func (q *query) Options() QueryOptions {
	return q.opts
}

// String prints routing table query in human readable form
func (q query) String() string {
	// this will help us build routing table string
	sb := &strings.Builder{}

	// create nice table printing structure
	table := tablewriter.NewWriter(sb)
	table.SetHeader([]string{"Service", "Gateway", "Network", "Policy"})

	strQuery := []string{
		q.opts.Service,
		q.opts.Gateway,
		q.opts.Network,
		fmt.Sprintf("%s", q.opts.Policy),
	}
	table.Append(strQuery)

	// render table into sb
	table.Render()

	return sb.String()
}
