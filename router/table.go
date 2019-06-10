package router

import (
	"errors"
	"fmt"
	"hash"
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
	Add(Route) error
	// Remove removes route from the table
	Remove(Route) error
	// Update updates route in the table
	Update(...RouteOption) error
	// Lookup looks up routes in the table
	Lookup(Query) ([]Route, error)
	// Size returns the size of the table
	Size() int
	// String prints the routing table
	String() string
}

// table is routing table
// It maps service name to routes
type table struct {
	// m stores routing table map
	m map[uint64]Route
	// h is a hasher hashes route entries
	h hash.Hash64
	sync.RWMutex
}

// NewTable creates new routing table and returns it
func NewTable() Table {
	h := fnv.New64()
	h.Reset()

	return &table{
		m: make(map[uint64]Route),
		h: h,
	}
}

// Add adds new routing entry
func (t *table) Add(r Route) error {
	t.Lock()
	defer t.Unlock()

	sum := t.hash(r)

	if _, ok := t.m[sum]; !ok {
		t.m[sum] = r
		return nil
	}

	if _, ok := t.m[sum]; ok && r.Options().Policy == OverrideIfExists {
		t.m[sum] = r
		return nil
	}

	return ErrDuplicateRoute
}

// Remove removes entry from the routing table
func (t *table) Remove(r Route) error {
	t.Lock()
	defer t.Unlock()

	sum := t.hash(r)

	if _, ok := t.m[sum]; !ok {
		return ErrRouteNotFound
	}

	delete(t.m, sum)

	return nil
}

// Update updates routing entry
func (t *table) Update(opts ...RouteOption) error {
	t.Lock()
	defer t.Unlock()

	r := NewRoute(opts...)

	sum := t.hash(r)

	if _, ok := t.m[sum]; !ok {
		return ErrRouteNotFound
	}

	if _, ok := t.m[sum]; ok {
		t.m[sum] = r
		return nil
	}

	return ErrRouteNotFound
}

// Lookup looks up entry in the routing table
func (t *table) Lookup(q Query) ([]Route, error) {
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

	for _, route := range t.m {
		strRoute := []string{
			route.Options().DestAddr,
			route.Options().Hop.Address(),
			fmt.Sprintf("%d", route.Options().SrcAddr),
			fmt.Sprintf("%d", route.Options().Metric),
		}
		table.Append(strRoute)
	}

	return sb.String()
}

func (t *table) hash(r Route) uint64 {
	srcAddr := r.Options().SrcAddr
	destAddr := r.Options().DestAddr
	routerAddr := r.Options().Hop.Address()

	t.h.Reset()
	t.h.Write([]byte(srcAddr + destAddr + routerAddr))

	return t.h.Sum64()
}
