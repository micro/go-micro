package router

import "errors"

var (
	// ErrRouteNotFound is returned when no rout was found
	ErrRouteNotFound = errors.New("route not found")
)

// Entry is micro network routing table entry
type Entry struct {
	// NetID is micro network ID
	NetID string
	// Hop is the next route hop
	Hop Router
	// Metric is route cost metric
	Metric int
}

// Table is routing table
// It maps service name to routes
type Table struct {
	// m stores routing table map
	m map[string][]Entry
}

// NewTable creates new routing table and returns it
func NewTable() *Table {
	return &Table{
		m: make(map[string][]Entry),
	}
}

// TODO: Define lookup query interface
// Lookup looks up entry in the routing table
func (t *Table) Lookup() (*Entry, error) {
	return nil, nil
}

// Remove removes entry from the routing table
func (t *Table) Remove(e *Entry) error {
	return nil
}

// Update updates routin entry
func (t *Table) Update(e *Entry) error {
	return nil
}

// Size returns the size of the routing table
func (t *Table) Size() int {
	return 0
}

// String returns text representation of routing table
func (t *Table) String() string {
	return ""
}
