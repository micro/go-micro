package router

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/gossip"
)

type router struct {
	opts  Options
	goss  registry.Registry
	table Table
	id    uuid.UUID
}

func newRouter(opts ...Option) Router {
	// TODO: figure out how to supply gossip registry options
	r := &router{
		goss:  gossip.NewRegistry(),
		table: NewTable(),
		id:    uuid.New(),
	}

	for _, o := range opts {
		o(&r.opts)
	}

	// TODO: need to start some gossip.Registry watch here

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

// Add adds new entry into routing table with given options.
// It returns error if the entry could not be added.
func (r *router) Add(e Entry) error {
	return r.table.Add(e)
}

// Remove removes entry from the routing table.
// It returns error if either the entry could not be removed or it does not exist.
func (r *router) Remove(e Entry) error {
	return r.table.Remove(e)
}

// Update updates an entry in the router's routing table
// It returns error if the entry was not found or it failed to be updated.
func (r *router) Update(opts ...EntryOption) error {
	return r.table.Update(opts...)
}

// Lookup makes a query lookup and returns all matching entries
func (r *router) Lookup(q Query) ([]*Entry, error) {
	return nil, ErrNotImplemented
}

// Table returns routing table
func (r *router) Table() Table {
	return r.table
}

// Network returns router's micro network
func (r *router) Network() string {
	return r.opts.Network
}

// Address returns router's bind address
func (r *router) Address() string {
	return r.opts.Address
}

// String prints debugging information about router
func (r *router) String() string {
	sb := &strings.Builder{}

	s := fmt.Sprintf("Router ID: %s\n", r.id.String())
	sb.WriteString(s)

	s = fmt.Sprintf("Router Local Address: %s\n", r.opts.Address)
	sb.WriteString(s)

	s = fmt.Sprintf("Router Network Address: %s\n", r.opts.Network)
	sb.WriteString(s)

	s = fmt.Sprintf("Routing table size: %d\n", r.opts.Table.Size())
	sb.WriteString(s)

	return sb.String()
}
