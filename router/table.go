package router

import (
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"strings"
	"sync"

	"github.com/google/uuid"
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
	// Remove removes existing route from the table
	Remove(Route) error
	// Update updates route in the table
	Update(...RouteOption) error
	// Lookup looks up routes in the table
	Lookup(Query) ([]Route, error)
	// Watch returns a watcher which allows you to track updates to the table
	Watch(opts ...WatchOption) (Watcher, error)
	// Size returns the size of the table
	Size() int
	// String prints the routing table
	String() string
}

// table is routing table
type table struct {
	// m stores routing table map
	m map[string]map[uint64]Route
	// h hashes route entries
	h hash.Hash64
	// w is a list of table watchers
	w map[string]*tableWatcher
	sync.RWMutex
}

// NewTable creates new routing table and returns it
func NewTable() Table {
	h := fnv.New64()
	h.Reset()

	return &table{
		m: make(map[string]map[uint64]Route),
		w: make(map[string]*tableWatcher),
		h: h,
	}
}

// Add adds a route to the routing table
func (t *table) Add(r Route) error {
	t.Lock()
	defer t.Unlock()

	destAddr := r.Options().DestAddr
	sum := t.hash(r)

	if _, ok := t.m[destAddr]; !ok {
		t.m[destAddr] = make(map[uint64]Route)
		t.m[destAddr][sum] = r
		go t.sendResult(&Result{Action: "add", Route: r})
		return nil
	}

	if _, ok := t.m[destAddr][sum]; ok && r.Options().Policy == OverrideIfExists {
		t.m[destAddr][sum] = r
		go t.sendResult(&Result{Action: "update", Route: r})
		return nil
	}

	return ErrDuplicateRoute
}

// Remove removes the route from the routing table
func (t *table) Remove(r Route) error {
	t.Lock()
	defer t.Unlock()

	destAddr := r.Options().DestAddr
	sum := t.hash(r)

	if _, ok := t.m[destAddr]; !ok {
		return ErrRouteNotFound
	}

	delete(t.m[destAddr], sum)
	go t.sendResult(&Result{Action: "remove", Route: r})

	return nil
}

// Update updates routing table using propvided options
func (t *table) Update(opts ...RouteOption) error {
	t.Lock()
	defer t.Unlock()

	r := NewRoute(opts...)

	destAddr := r.Options().DestAddr
	sum := t.hash(r)

	if _, ok := t.m[destAddr]; !ok {
		return ErrRouteNotFound
	}

	if _, ok := t.m[destAddr][sum]; ok {
		t.m[destAddr][sum] = r
		go t.sendResult(&Result{Action: "update", Route: r})
		return nil
	}

	return ErrRouteNotFound
}

// Lookup queries routing table and returns all routes that match it
func (t *table) Lookup(q Query) ([]Route, error) {
	return nil, ErrNotImplemented
}

// Watch returns routing table entry watcher
func (t *table) Watch(opts ...WatchOption) (Watcher, error) {
	// by default watch everything
	wopts := WatchOptions{
		DestAddr: "*",
		Network:  "*",
	}

	for _, o := range opts {
		o(&wopts)
	}

	watcher := &tableWatcher{
		opts:    wopts,
		resChan: make(chan *Result, 10),
		done:    make(chan struct{}),
	}

	t.Lock()
	t.w[uuid.New().String()] = watcher
	t.Unlock()

	return watcher, nil
}

// sendResult sends rules to all subscribe watchers
func (t *table) sendResult(r *Result) {
	t.RLock()
	defer t.RUnlock()

	for _, w := range t.w {
		select {
		case w.resChan <- r:
		case <-w.done:
		}
	}
}

// Size returns the size of the routing table
func (t *table) Size() int {
	t.RLock()
	defer t.RUnlock()

	return len(t.m)
}

// String returns debug information
func (t *table) String() string {
	t.RLock()
	defer t.RUnlock()

	// this will help us build routing table string
	sb := &strings.Builder{}

	// create nice table printing structure
	table := tablewriter.NewWriter(sb)
	table.SetHeader([]string{"Destination", "Gateway", "Network", "Metric"})

	for _, destRoute := range t.m {
		for _, route := range destRoute {
			strRoute := []string{
				route.Options().DestAddr,
				route.Options().Gateway.Address(),
				route.Options().Gateway.Network(),
				fmt.Sprintf("%d", route.Options().Metric),
			}
			table.Append(strRoute)
		}
	}

	// render table into sb
	table.Render()

	return sb.String()
}

// hash hashes the route using router gateway and network address
func (t *table) hash(r Route) uint64 {
	gwAddr := r.Options().Gateway.Address()
	netAddr := r.Options().Network

	t.h.Reset()
	t.h.Write([]byte(gwAddr + netAddr))

	return t.h.Sum64()
}
