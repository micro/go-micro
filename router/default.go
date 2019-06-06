package router

import (
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/gossip"
)

type router struct {
	opts Options
	goss registry.Registry
	t    *Table
}

func newRouter(opts ...Option) Router {
	// TODO: figure out how to supply gossip registry options
	r := &router{
		goss: gossip.NewRegistry(),
		t:    NewTable(),
	}

	for _, o := range opts {
		o(&r.opts)
	}

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
func (r *router) Add(e *Entry, opts ...RouteOption) error {
	return nil
}

// Remove removes entry from the routing table.
// It returns error if either the entry could not be removed or it does not exist.
func (r *router) Remove(e *Entry) error {
	return nil
}

// Update updates an entry in the router's routing table
// It returns error if the entry was not found or it failed to be updated.
func (r *router) Update(e *Entry) error {
	return nil
}

// Lookup makes a query lookup and returns all matching entries
func (r *router) Lookup(q Query) ([]*Entry, error) {
	return nil, nil
}

// Table returns routing table
func (r *router) Table() *Table {
	return nil
}

// Address returns router's network address
func (r *router) Address() string {
	return ""
}

// String prints debugging information about router
func (r *router) String() string {
	return ""
}
