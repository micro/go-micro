package registry

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/router"
)

// table is an in-memory routing table
type table struct {
	sync.RWMutex
	// lookup for a service
	lookup func(string) ([]router.Route, error)
	// routes stores service routes
	routes map[string]map[uint64]*route
	// watchers stores table watchers
	watchers map[string]*tableWatcher
}

type route struct {
	route   router.Route
	updated time.Time
}

// newtable creates a new routing table and returns it
func newTable(lookup func(string) ([]router.Route, error), opts ...router.Option) *table {
	return &table{
		lookup:   lookup,
		routes:   make(map[string]map[uint64]*route),
		watchers: make(map[string]*tableWatcher),
	}
}

// pruneRoutes will prune routes older than the time specified
func (t *table) pruneRoutes(olderThan time.Duration) {
	var routes []router.Route

	t.Lock()

	// search for all the routes
	for _, routeList := range t.routes {
		for _, r := range routeList {
			// if any route is older than
			if time.Since(r.updated).Seconds() > olderThan.Seconds() {
				routes = append(routes, r.route)
			}
		}
	}

	t.Unlock()

	// delete the routes we've found
	for _, route := range routes {
		t.Delete(route)
	}
}

// deleteService removes the entire service
func (t *table) deleteService(service, network string) {
	t.Lock()
	defer t.Unlock()

	routes, ok := t.routes[service]
	if !ok {
		return
	}

	// delete the routes for the service
	for hash, rt := range routes {
		// TODO: check if this causes a problem
		// with * in the network if that is a thing
		// or blank strings
		if rt.route.Network != network {
			continue
		}
		delete(routes, hash)
	}

	// delete the map for the service if its empty
	if len(routes) == 0 {
		delete(t.routes, service)
		return
	}

	// save the routes
	t.routes[service] = routes
}

// sendEvent sends events to all subscribed watchers
func (t *table) sendEvent(e *router.Event) {
	t.RLock()
	defer t.RUnlock()

	if len(e.Id) == 0 {
		e.Id = uuid.New().String()
	}

	for _, w := range t.watchers {
		select {
		case w.resChan <- e:
		case <-w.done:
		// don't block forever
		case <-time.After(time.Second):
		}
	}
}

// Create creates new route in the routing table
func (t *table) Create(r router.Route) error {
	service := r.Service
	sum := r.Hash()

	t.Lock()
	defer t.Unlock()

	// check if there are any routes in the table for the route destination
	if _, ok := t.routes[service]; !ok {
		t.routes[service] = make(map[uint64]*route)
	}

	// add new route to the table for the route destination
	if _, ok := t.routes[service][sum]; ok {
		return router.ErrDuplicateRoute
	}

	// create the route
	t.routes[service][sum] = &route{r, time.Now()}

	if logger.V(logger.DebugLevel, logger.DefaultLogger) {
		logger.Debugf("Router emitting %s for route: %s", router.Create, r.Address)
	}

	// send a route created event
	go t.sendEvent(&router.Event{Type: router.Create, Timestamp: time.Now(), Route: r})

	return nil
}

// Delete deletes the route from the routing table
func (t *table) Delete(r router.Route) error {
	service := r.Service
	sum := r.Hash()

	t.Lock()
	defer t.Unlock()

	if _, ok := t.routes[service]; !ok {
		return router.ErrRouteNotFound
	}

	if _, ok := t.routes[service][sum]; !ok {
		return router.ErrRouteNotFound
	}

	// delete the route from the service
	delete(t.routes[service], sum)

	// delete the whole map if there are no routes left
	if len(t.routes[service]) == 0 {
		delete(t.routes, service)
	}

	if logger.V(logger.DebugLevel, logger.DefaultLogger) {
		logger.Debugf("Router emitting %s for route: %s", router.Delete, r.Address)
	}
	go t.sendEvent(&router.Event{Type: router.Delete, Timestamp: time.Now(), Route: r})

	return nil
}

// Update updates routing table with the new route
func (t *table) Update(r router.Route) error {
	service := r.Service
	sum := r.Hash()

	t.Lock()
	defer t.Unlock()

	// check if the route destination has any routes in the table
	if _, ok := t.routes[service]; !ok {
		t.routes[service] = make(map[uint64]*route)
	}

	if _, ok := t.routes[service][sum]; !ok {
		// update the route
		t.routes[service][sum] = &route{r, time.Now()}

		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Router emitting %s for route: %s", router.Update, r.Address)
		}
		go t.sendEvent(&router.Event{Type: router.Update, Timestamp: time.Now(), Route: r})
		return nil
	}

	// just update the route, but dont emit Update event
	t.routes[service][sum] = &route{r, time.Now()}

	return nil
}

// List returns a list of all routes in the table
func (t *table) List() ([]router.Route, error) {
	t.RLock()
	defer t.RUnlock()

	var routes []router.Route
	for _, rmap := range t.routes {
		for _, route := range rmap {
			routes = append(routes, route.route)
		}
	}

	return routes, nil
}

// isMatch checks if the route matches given query options
func isMatch(route router.Route, address, gateway, network, rtr, link string) bool {
	// matches the values provided
	match := func(a, b string) bool {
		if a == "*" || b == "*" || a == b {
			return true
		}
		return false
	}

	// a simple struct to hold our values
	type compare struct {
		a string
		b string
	}

	// compare the following values
	values := []compare{
		{gateway, route.Gateway},
		{network, route.Network},
		{rtr, route.Router},
		{address, route.Address},
		{link, route.Link},
	}

	for _, v := range values {
		// attempt to match each value
		if !match(v.a, v.b) {
			return false
		}
	}

	return true
}

// filterRoutes finds all the routes for given network and router and returns them
func filterRoutes(routes map[uint64]*route, opts router.QueryOptions) []router.Route {
	address := opts.Address
	gateway := opts.Gateway
	network := opts.Network
	rtr := opts.Router
	link := opts.Link

	// routeMap stores the routes we're going to advertise
	routeMap := make(map[string][]router.Route)

	for _, rt := range routes {
		// get the actual route
		route := rt.route

		if isMatch(route, address, gateway, network, rtr, link) {
			// add matchihg route to the routeMap
			routeKey := route.Service + "@" + route.Network
			routeMap[routeKey] = append(routeMap[routeKey], route)
		}
	}

	var results []router.Route

	for _, route := range routeMap {
		results = append(results, route...)
	}

	return results
}

// Lookup queries routing table and returns all routes that match the lookup query
func (t *table) Query(q ...router.QueryOption) ([]router.Route, error) {
	// create new query options
	opts := router.NewQuery(q...)

	// create a cwslicelist of query results
	results := make([]router.Route, 0, len(t.routes))

	// readAndFilter routes for this service under read lock.
	readAndFilter := func(q router.QueryOptions) ([]router.Route, bool) {
		t.RLock()
		defer t.RUnlock()

		routes, ok := t.routes[q.Service]
		if !ok || len(routes) == 0 {
			return nil, false
		}

		return filterRoutes(routes, q), true
	}

	if opts.Service != "*" {
		// try and load services from the cache
		if routes, ok := readAndFilter(opts); ok {
			return routes, nil
		}

		// lookup the route and try again
		// TODO: move this logic out of the hot path
		// being hammered on queries will require multiple lookups
		routes, err := t.lookup(opts.Service)
		if err != nil {
			return nil, err
		}

		// cache the routes
		for _, rt := range routes {
			t.Create(rt)
		}

		// try again
		if routes, ok := readAndFilter(opts); ok {
			return routes, nil
		}

		return nil, router.ErrRouteNotFound
	}

	// search through all destinations
	t.RLock()

	for _, routes := range t.routes {
		// filter the routes
		found := filterRoutes(routes, opts)
		// ensure we don't append zero length routes
		if len(found) == 0 {
			continue
		}
		results = append(results, found...)
	}

	t.RUnlock()

	return results, nil
}

// Watch returns routing table entry watcher
func (t *table) Watch(opts ...router.WatchOption) (router.Watcher, error) {
	// by default watch everything
	wopts := router.WatchOptions{
		Service: "*",
	}

	for _, o := range opts {
		o(&wopts)
	}

	w := &tableWatcher{
		id:      uuid.New().String(),
		opts:    wopts,
		resChan: make(chan *router.Event, 10),
		done:    make(chan struct{}),
	}

	// when the watcher is stopped delete it
	go func() {
		<-w.done
		t.Lock()
		delete(t.watchers, w.id)
		t.Unlock()
	}()

	// save the watcher
	t.Lock()
	t.watchers[w.id] = w
	t.Unlock()

	return w, nil
}
