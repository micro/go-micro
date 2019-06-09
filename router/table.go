package router

import (
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"

	"github.com/olekukonko/tablewriter"
)

var (
	// ErrRouteNotFound is returned when no route was found
	ErrRouteNotFound = errors.New("route not found")
	// ErrDuplicateRoute is return when route already exists
	ErrDuplicateRoute = errors.New("duplicate route")
	// ErrNotImplemented is returned when some functionality has not been implemented
	ErrNotImplemented = errors.New("not implemented")
)

// Table is routing table
type Table interface {
	// Add adds new route to the table
	Add(Entry) error
	// Remove removes route from the table
	Remove(Entry) error
	// Update updates route in the table
	Update(...EntryOption) error
	// Lookup looks up routes in the table
	Lookup(Query) ([]Entry, error)
	// Size returns the size of the table
	Size() int
	// String prints the routing table
	String() string
}

// table is routing table
// It maps service name to routes
type table struct {
	// m stores routing table map
	m map[string]map[uint64]Entry
	sync.RWMutex
}

// NewTable creates new routing table and returns it
func NewTable() Table {
	return &table{
		m: make(map[string]map[uint64]Entry),
	}
}

// Add adds new routing entry
func (t *table) Add(e Entry) error {
	t.Lock()
	defer t.Unlock()

	destAddr := e.Options().DestAddr
	h := fnv.New64()
	h.Write([]byte(e.Options().DestAddr + e.Options().Hop.Address()))

	if _, ok := t.m[destAddr]; !ok {
		// create new map for DestAddr routes
		t.m[destAddr] = make(map[uint64]Entry)
		t.m[destAddr][h.Sum64()] = e
		return nil
	}

	if _, ok := t.m[destAddr][h.Sum64()]; ok && e.Options().Policy == OverrideIfExists {
		t.m[destAddr][h.Sum64()] = e
		return nil
	}

	return ErrDuplicateRoute
}

// Remove removes entry from the routing table
func (t *table) Remove(e Entry) error {
	t.Lock()
	defer t.Unlock()

	destAddr := e.Options().DestAddr
	h := fnv.New64()
	h.Write([]byte(e.Options().DestAddr + e.Options().Hop.Address()))

	if _, ok := t.m[destAddr]; !ok {
		return ErrRouteNotFound
	} else {
		delete(t.m[destAddr], h.Sum64())
		return nil
	}

	return nil
}

// Update updates routing entry
func (t *table) Update(opts ...EntryOption) error {
	t.Lock()
	defer t.Unlock()

	e := NewEntry(opts...)

	destAddr := e.Options().DestAddr
	h := fnv.New64()
	h.Write([]byte(e.Options().DestAddr + e.Options().Hop.Address()))

	if _, ok := t.m[destAddr]; !ok {
		return ErrRouteNotFound
	}

	if _, ok := t.m[destAddr][h.Sum64()]; ok {
		t.m[destAddr][h.Sum64()] = e
		return nil
	}

	return ErrRouteNotFound
}

// Lookup looks up entry in the routing table
func (t *table) Lookup(q Query) ([]Entry, error) {
	return nil, ErrNotImplemented
}

// Size returns the size of the routing table
func (t *table) Size() int {
	t.RLock()
	defer t.RUnlock()

	return len(t.m)
}

// String returns text representation of routing table
func (t *table) String() string {
	t.RLock()
	defer t.RUnlock()

	// this will help us build routing table string
	sb := &strings.Builder{}

	// create nice table printing structure
	table := tablewriter.NewWriter(sb)
	table.SetHeader([]string{"Dest", "Hop", "Src", "Metric"})

	var destAddr, prevAddr string

	for _, entries := range t.m {
		for _, entry := range entries {
			destAddr = entry.Options().DestAddr
			// we want to avoid printing the same dest address
			if prevAddr == destAddr {
				destAddr = ""
			}
			strEntry := []string{
				destAddr,
				entry.Options().Hop.Address(),
				fmt.Sprintf("%d", entry.Options().SrcAddr),
				fmt.Sprintf("%d", entry.Options().Metric),
			}
			table.Append(strEntry)
			prevAddr = destAddr
		}
	}

	return sb.String()
}
