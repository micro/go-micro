package router

import (
	"fmt"
	"strings"

	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/gossip"
	"github.com/olekukonko/tablewriter"
)

type router struct {
	opts Options
	goss registry.Registry
}

func newRouter(opts ...Option) Router {
	// set default options
	options := Options{
		Table: NewTable(),
	}

	for _, o := range opts {
		o(&options)
	}

	goss := gossip.NewRegistry(
		gossip.Address(options.GossipAddr),
	)

	r := &router{
		opts: options,
		goss: goss,
	}

	// TODO: start gossip.Registry watch here

	return r
}

// Init initializes router with given options
func (r *router) Init(opts ...Option) error {
	for _, o := range opts {
		o(&r.opts)
	}
	return nil
}

// Options returns router options
func (r *router) Options() Options {
	return r.opts
}

// Table returns routing table
func (r *router) Table() Table {
	return r.opts.Table
}

// Address returns router's bind address
func (r *router) Address() string {
	return r.opts.Address
}

// Network returns router's micro network
func (r *router) Network() string {
	return r.opts.NetworkAddr
}

// String prints debugging information about router
func (r *router) String() string {
	sb := &strings.Builder{}

	table := tablewriter.NewWriter(sb)
	table.SetHeader([]string{"ID", "Address", "Gossip", "Network", "Table"})

	data := []string{
		r.opts.ID,
		r.opts.Address,
		r.opts.GossipAddr,
		r.opts.NetworkAddr,
		fmt.Sprintf("%d", r.opts.Table.Size()),
	}
	table.Append(data)

	// render table into sb
	table.Render()

	return sb.String()
}
