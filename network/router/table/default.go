package table

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// TableOptions specify routing table options
// TODO: table options TBD in the future
type TableOptions struct{}

// table is an in memory routing table
type table struct {
	// opts are table options
	opts TableOptions
	// m stores routing table map
	m map[string]map[uint64]Route
	// w is a list of table watchers
	w map[string]*tableWatcher
	sync.RWMutex
}

// newTable creates a new routing table and returns it
func newTable(opts ...TableOption) Table {
	// default options
	var options TableOptions

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	return &table{
		opts: options,
		m:    make(map[string]map[uint64]Route),
		w:    make(map[string]*tableWatcher),
	}
}

// Init initializes routing table with options
func (t *table) Init(opts ...TableOption) error {
	for _, o := range opts {
		o(&t.opts)
	}
	return nil
}

// Options returns routing table options
func (t *table) Options() TableOptions {
	return t.opts
}

// Create creates new route in the routing table
func (t *table) Create(r Route) error {
	service := r.Service
	sum := r.Hash()

	t.Lock()
	defer t.Unlock()

	// check if there are any routes in the table for the route destination
	if _, ok := t.m[service]; !ok {
		t.m[service] = make(map[uint64]Route)
		t.m[service][sum] = r
		go t.sendEvent(&Event{Type: Create, Timestamp: time.Now(), Route: r})
		return nil
	}

	// add new route to the table for the route destination
	if _, ok := t.m[service][sum]; !ok {
		t.m[service][sum] = r
		go t.sendEvent(&Event{Type: Create, Timestamp: time.Now(), Route: r})
		return nil
	}

	return ErrDuplicateRoute
}

// Delete deletes the route from the routing table
func (t *table) Delete(r Route) error {
	service := r.Service
	sum := r.Hash()

	t.Lock()
	defer t.Unlock()

	if _, ok := t.m[service]; !ok {
		return ErrRouteNotFound
	}

	delete(t.m[service], sum)
	go t.sendEvent(&Event{Type: Delete, Timestamp: time.Now(), Route: r})

	return nil
}

// Update updates routing table with the new route
func (t *table) Update(r Route) error {
	service := r.Service
	sum := r.Hash()

	t.Lock()
	defer t.Unlock()

	// check if the route destination has any routes in the table
	if _, ok := t.m[service]; !ok {
		return ErrRouteNotFound
	}

	// if the route has been found update it
	if _, ok := t.m[service][sum]; ok {
		t.m[service][sum] = r
		go t.sendEvent(&Event{Type: Update, Timestamp: time.Now(), Route: r})
		return nil
	}

	return ErrRouteNotFound
}

// List returns a list of all routes in the table
func (t *table) List() ([]Route, error) {
	t.RLock()
	defer t.RUnlock()

	var routes []Route
	for _, rmap := range t.m {
		for _, route := range rmap {
			routes = append(routes, route)
		}
	}

	return routes, nil
}

// isMatch checks if the route matches given network and router
func isMatch(route Route, network, router string) bool {
	if network == "*" || network == route.Network {
		if router == "*" || router == route.Gateway {
			return true
		}
	}
	return false
}

// findRoutes finds all the routes for given network and router and returns them
func findRoutes(routes map[uint64]Route, network, router string) []Route {
	var results []Route
	for _, route := range routes {
		if isMatch(route, network, router) {
			results = append(results, route)
		}
	}
	return results
}

// Lookup queries routing table and returns all routes that match the lookup query
func (t *table) Lookup(q Query) ([]Route, error) {
	t.RLock()
	defer t.RUnlock()

	if q.Options().Service != "*" {
		// no routes found for the destination and query policy is not a DiscardIfNone
		if _, ok := t.m[q.Options().Service]; !ok && q.Options().Policy != DiscardIfNone {
			return nil, ErrRouteNotFound
		}
		return findRoutes(t.m[q.Options().Service], q.Options().Network, q.Options().Gateway), nil
	}

	var results []Route
	// search through all destinations
	for _, routes := range t.m {
		results = append(results, findRoutes(routes, q.Options().Network, q.Options().Gateway)...)
	}

	return results, nil
}

// Watch returns routing table entry watcher
func (t *table) Watch(opts ...WatchOption) (Watcher, error) {
	// by default watch everything
	wopts := WatchOptions{
		Service: "*",
	}

	for _, o := range opts {
		o(&wopts)
	}

	watcher := &tableWatcher{
		opts:    wopts,
		resChan: make(chan *Event, 10),
		done:    make(chan struct{}),
	}

	t.Lock()
	t.w[uuid.New().String()] = watcher
	t.Unlock()

	return watcher, nil
}

// sendEvent sends rules to all subscribe watchers
func (t *table) sendEvent(r *Event) {
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

	size := 0
	for dest, _ := range t.m {
		size += len(t.m[dest])
	}

	return size
}

// String returns debug information
func (t table) String() string {
	return "table"
}
