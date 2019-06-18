package router

import (
	"fmt"
	"hash"
	"hash/fnv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/micro/go-log"
	"github.com/olekukonko/tablewriter"
)

// TODO: table options TBD in the future
// TableOptions are routing table options
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

	log.Logf("[table] AddRoute request %d %s: \n%s", sum, r.Options().Policy, r)

	// check if the destination has any routes in the table
	if _, ok := t.m[destAddr]; !ok {
		log.Logf("[table] destination does NOT exist ADDING: \n%s", r)
		t.m[destAddr] = make(map[uint64]Route)
		t.m[destAddr][sum] = r
		go t.sendEvent(&Event{Type: CreateEvent, Route: r})
		return nil
	}

	// add new route to the table for the given destination
	if _, ok := t.m[destAddr][sum]; !ok {
		log.Logf("[table] route does NOT exist ADDING: \n%s", r)
		t.m[destAddr][sum] = r
		go t.sendEvent(&Event{Type: CreateEvent, Route: r})
		return nil
	}

	// only add the route if it exists and if override is requested
	if _, ok := t.m[destAddr][sum]; ok && r.Options().Policy == OverrideIfExists {
		log.Logf("[table] route does exist OVERRIDING: \n%s", r)
		t.m[destAddr][sum] = r
		go t.sendEvent(&Event{Type: UpdateEvent, Route: r})
		return nil
	}

	// if we reached this point without already returning the route already exists
	// we return nil only if explicitly requested by the client
	if r.Options().Policy == IgnoreIfExists {
		log.Logf("[table] route does exist IGNORING: \n%s", r)
		return nil
	}

	log.Logf("[table] AddRoute request: DUPPLICATE ROUTE")

	return ErrDuplicateRoute
}

// Delete deletes the route from the routing table
func (t *table) Delete(r Route) error {
	t.Lock()
	defer t.Unlock()

	destAddr := r.Options().DestAddr
	sum := t.hash(r)

	log.Logf("[table] DeleteRoute request %d: \n%s", sum, r)

	if _, ok := t.m[destAddr]; !ok {
		log.Logf("[table] DeleteRoute Route NOT found: %s", r)
		return ErrRouteNotFound
	}

	delete(t.m[destAddr], sum)
	go t.sendEvent(&Event{Type: DeleteEvent, Route: r})

	return nil
}

// Update updates routing table with new route
func (t *table) Update(r Route) error {
	destAddr := r.Options().DestAddr
	sum := t.hash(r)

	t.Lock()
	defer t.Unlock()

	// check if the destAddr has ANY routes in the table
	if _, ok := t.m[destAddr]; !ok {
		return ErrRouteNotFound
	}

	// if the route has been found update it
	if _, ok := t.m[destAddr][sum]; ok {
		t.m[destAddr][sum] = r
		go t.sendEvent(&Event{Type: UpdateEvent, Route: r})
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
				route.Options().Network,
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
	destAddr := r.Options().DestAddr
	gwAddr := r.Options().Gateway.Address()
	netAddr := r.Options().Network

	t.h.Reset()
	t.h.Write([]byte(destAddr + gwAddr + netAddr))

	return t.h.Sum64()
}
