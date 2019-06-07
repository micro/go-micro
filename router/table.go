package router

import (
	"errors"
	"sync"
)

var (
	// DefaultRouter returns default micro router
	DefaultTable = NewTable()
	// ErrRouteNotFound is returned when no route was found
	ErrRouteNotFound = errors.New("route not found")
	// ErrDuplicateRoute is return when route already exists
	ErrDuplicateRoute = errors.New("duplicate route")
)

// Table is routing table
type Table interface {
	// Add adds new route to the table
	Add(*Entry) error
	// Remove removes route from the table
	Remove(*Entry) error
	// Update updates route in the table
	Update(*Entry) error
	// Lookup looks up routes in the table
	Lookup(Query) ([]*Entry, error)
	// Size returns the size of the table
	Size() int
	// String prints the routing table
	String() string
}

// Entry is micro network routing table entry
type Entry struct {
	// Addr is destination address
	Addr string
	// NetID is micro network ID
	NetID string
	// Hop is the next route hop
	Hop Router
	// Metric is route cost metric
	Metric int
}

// table is routing table
// It maps service name to routes
type table struct {
	// m stores routing table map
	m map[string][]Entry
	sync.RWMutex
}

// NewTable creates new routing table and returns it
func NewTable() Table {
	return &table{
		m: make(map[string][]Entry),
	}
}

// Add adds new routing entry
func (t *table) Add(e *Entry) error {
	return nil
}

// Remove removes entry from the routing table
func (t *table) Remove(e *Entry) error {
	return nil
}

// Update updates routin entry
func (t *table) Update(e *Entry) error {
	return nil
}

// Lookup looks up entry in the routing table
func (t *table) Lookup(q Query) ([]*Entry, error) {
	return nil, nil
}

// Size returns the size of the routing table
func (t *table) Size() int {
	return len(t.m)
}

// String returns text representation of routing table
func (t *table) String() string {
	return ""
}
