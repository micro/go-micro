package router

import (
	"fmt"
	"hash"
	"hash/fnv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
)

// TableOptions are routing table options
// TODO: This will allow for arbitrary routing table options in the future
type TableOptions struct{}

// table is in memory routing table
type table struct {
	// opts are table options
	opts TableOptions
	// m stores routing table map
	m map[string]map[uint64]Route
	// h hashes route entries
	h hash.Hash64
	// w is a list of table watchers
	w map[string]*tableWatcher
	sync.RWMutex
}

// newTable creates in memory routing table and returns it
func newTable(opts ...TableOption) Table {
	// default options
	var options TableOptions

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	h := fnv.New64()
	h.Reset()

	return &table{
		opts: options,
		m:    make(map[string]map[uint64]Route),
		w:    make(map[string]*tableWatcher),
		h:    h,
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

// Add adds a route to the routing table
func (t *table) Add(r Route) error {
	destAddr := r.Options().DestAddr
	sum := t.hash(r)

	t.Lock()
	defer t.Unlock()

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

	if r.Options().Policy == IgnoreIfExists {
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

// Update updates routing table with new route
func (t *table) Update(r Route) error {
	destAddr := r.Options().DestAddr
	sum := t.hash(r)

	t.Lock()
	defer t.Unlock()

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
	t.RLock()
	defer t.RUnlock()

	var results []Route

	for destAddr, routes := range t.m {
		if q.Options().DestAddr != "*" {
			if q.Options().DestAddr != destAddr {
				continue
			}
			for _, route := range routes {
				if q.Options().Network == "*" || q.Options().Network == route.Options().Network {
					if q.Options().Gateway.ID() == "*" || q.Options().Gateway.ID() == route.Options().Gateway.ID() {
						results = append(results, route)
					}
				}
			}
		}

		if q.Options().DestAddr == "*" {
			for _, route := range routes {
				if q.Options().Network == "*" || q.Options().Network == route.Options().Network {
					if q.Options().Gateway.ID() == "*" || q.Options().Gateway.ID() == route.Options().Gateway.ID() {
						results = append(results, route)
					}
				}
			}
		}
	}

	if len(results) == 0 && q.Options().Policy != DiscardNoRoute {
		return nil, ErrRouteNotFound
	}

	return results, nil
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
