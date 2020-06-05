package router

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/registry"
)

var (
	// AdvertiseEventsTick is time interval in which the router advertises route updates
	AdvertiseEventsTick = 10 * time.Second
	// DefaultAdvertTTL is default advertisement TTL
	DefaultAdvertTTL = 2 * time.Minute
)

// router implements default router
type router struct {
	sync.RWMutex

	table         *table
	options       Options
	exit          chan bool
	eventChan     chan *Event
	networks      map[string]bool
	resetWatchers map[string]chan bool

	// advert subscribers
	sub         sync.RWMutex
	subscribers map[string]chan *Advert
}

// newRouter creates new router and returns it
func newRouter(opts ...Option) Router {
	// get default options
	options := DefaultOptions()

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// construct the router
	r := &router{
		networks:      map[string]bool{options.Network: true},
		options:       options,
		exit:          make(chan bool),
		table:         newTable(options),
		subscribers:   make(map[string]chan *Advert),
		resetWatchers: make(map[string]chan bool),
	}

	// load routes from the registry and then watch them async
	r.populateRoutes(r.options.Registry, options.Network)
	go r.watchRegistry(options.Network)

	return r
}

// Init initializes router with given options
func (r *router) Init(opts ...Option) error {
	r.Lock()
	defer r.Unlock()

	for _, o := range opts {
		o(&r.options)
	}

	// the underlying registry can change so we reset the router table, repopulate the caches for every
	// network we're following and then instruct all the registry watchers to reset
	r.table = newTable(r.options)

	for _, w := range r.resetWatchers {
		w <- true
	}

	for network := range r.networks {
		if err := r.populateRoutes(r.options.Registry, network); err != nil {
			return err
		}
	}

	return nil
}

// Options returns router options
func (r *router) Options() Options {
	r.RLock()
	defer r.RUnlock()
	options := r.options
	return options
}

// Table returns routing table
func (r *router) Table() Table {
	return r.table
}

// manageRoute applies action on a given route
func (r *router) manageRoute(route Route, action string) error {
	switch action {
	case "create":
		if err := r.table.Create(route); err != nil && err != ErrDuplicateRoute {
			return fmt.Errorf("failed adding route for service %s: %s", route.Service, err)
		}
	case "delete":
		if err := r.table.Delete(route); err != nil && err != ErrRouteNotFound {
			return fmt.Errorf("failed deleting route for service %s: %s", route.Service, err)
		}
	case "update":
		if err := r.table.Update(route); err != nil {
			return fmt.Errorf("failed updating route for service %s: %s", route.Service, err)
		}
	default:
		return fmt.Errorf("failed to manage route for service %s: unknown action %s", route.Service, action)
	}

	return nil
}

// manageServiceRoutes applies action to all routes of the service.
// It returns error of the action fails with error.
func (r *router) manageRoutes(service *registry.Service, action string, network string) error {
	// action is the routing table action
	action = strings.ToLower(action)

	// take route action on each service node
	for _, node := range service.Nodes {
		route := Route{
			Service: service.Name,
			Address: node.Address,
			Gateway: "",
			Network: network,
			Router:  r.options.Id,
			Link:    DefaultLink,
			Metric:  DefaultLocalMetric,
		}

		if err := r.manageRoute(route, action); err != nil {
			return err
		}
	}

	return nil
}

// populareRoutes applies the create action to all routes of each service found in the registry.
// It returns error if either the services failed to be listed or the routing table action fails.
func (r *router) populateRoutes(reg registry.Registry, domain string) error {
	// add all local service routes into the routing table
	services, err := reg.ListServices(registry.ListDomain(domain))
	if err != nil {
		return fmt.Errorf("failed listing services: %v", err)
	}

	// add each service node as a separate route
	for _, service := range services {
		// get the service to retrieve all its info
		srvs, err := reg.GetService(service.Name, registry.GetDomain(domain))
		if err != nil {
			continue
		}
		// manage the routes for all returned services
		for _, srv := range srvs {
			if err := r.manageRoutes(srv, "create", domain); err != nil {
				return err
			}
		}
	}

	// add default gateway into routing table
	if r.options.Gateway != "" {
		// note, the only non-default value is the gateway
		route := Route{
			Service: "*",
			Address: "*",
			Gateway: r.options.Gateway,
			Network: "*",
			Router:  r.options.Id,
			Link:    DefaultLink,
			Metric:  DefaultLocalMetric,
		}
		if err := r.table.Create(route); err != nil {
			return fmt.Errorf("failed adding default gateway route: %s", err)
		}
	}

	return nil
}

// watchRegistry updates the routing table based on the received events
func (r *router) watchRegistry(network string) {
	watch := func() error {
		w, err := r.options.Registry.Watch()
		if err != nil {
			return err
		}
		defer w.Stop()

		c := w.Chan()

		for {
			select {
			case <-r.exit:
				return nil
			case <-r.resetWatchers[network]:
				return registry.ErrWatcherStopped
			case res, ok := <-c:
				if !ok {
					return registry.ErrWatcherStopped
				}
				r.manageRoutes(res.Service, res.Action, network)
			}
		}
	}

	// we don't want to exist every time a registry watcher fails since this can happen for a range of
	// reasons, e.g. because the node was replaced. when the watcher errors, we'll create a new watcher
	// and try again.
	for {
		err := watch()
		// the watcher ended safely, exit
		if err == nil {
			return
		}

		// this error doesn't need to be reported
		if err == registry.ErrWatcherStopped {
			continue
		}

		// an unexpected error occured, log it and try again
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("failed creating registry watcher: %v", err)
		}
	}
}

// watchTable watches routing table entries and either adds or deletes locally registered service to/from network registry
// It returns error if the locally registered services either fails to be added/deleted to/from network registry.
func (r *router) watchTable(w Watcher) error {
	exit := make(chan bool)

	defer func() {
		close(exit)
	}()

	// wait in the background for the router to stop
	// when the router stops, stop the watcher and exit
	go func() {
		defer w.Stop()

		select {
		case <-r.exit:
			return
		case <-exit:
			return
		}
	}()

	for {
		event, err := w.Next()
		if err != nil {
			if err != ErrWatcherStopped {
				return err
			}
			break
		}

		select {
		case <-r.exit:
			close(r.eventChan)
			return nil
		case r.eventChan <- event:
			// process event
		}
	}

	return nil
}

// publishAdvert publishes router advert to advert channel
func (r *router) publishAdvert(advType AdvertType, events []*Event) {
	a := &Advert{
		Id:        r.options.Id,
		Type:      advType,
		TTL:       DefaultAdvertTTL,
		Timestamp: time.Now(),
		Events:    events,
	}

	r.sub.RLock()
	for _, sub := range r.subscribers {
		// now send the message
		select {
		case sub <- a:
		case <-r.exit:
			r.sub.RUnlock()
			return
		}
	}
	r.sub.RUnlock()
}

// adverts maintains a map of router adverts
type adverts map[uint64]*Event

// advertiseEvents advertises routing table events
// It suppresses unhealthy flapping events and advertises healthy events upstream.
func (r *router) advertiseEvents() error {
	// ticker to periodically scan event for advertising
	ticker := time.NewTicker(AdvertiseEventsTick)
	defer ticker.Stop()

	// adverts is a map of advert events
	adverts := make(adverts)

	// routing table watcher
	w, err := r.Watch()
	if err != nil {
		return err
	}
	defer w.Stop()

	go func() {
		var err error

		for {
			select {
			case <-r.exit:
				return
			default:
				if w == nil {
					// routing table watcher
					w, err = r.Watch()
					if err != nil {
						if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
							logger.Errorf("Error creating watcher: %v", err)
						}
						time.Sleep(time.Second)
						continue
					}
				}

				if err := r.watchTable(w); err != nil {
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						logger.Errorf("Error watching table: %v", err)
					}
					time.Sleep(time.Second)
				}

				if w != nil {
					// reset
					w.Stop()
					w = nil
				}
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			// If we're not advertising any events then sip processing them entirely
			if r.options.Advertise == AdvertiseNone {
				continue
			}

			var events []*Event

			// collect all events which are not flapping
			for key, event := range adverts {
				// if we only advertise local routes skip processing anything not link local
				if r.options.Advertise == AdvertiseLocal && event.Route.Link != "local" {
					continue
				}

				// copy the event and append
				e := new(Event)
				// this is ok, because router.Event only contains builtin types
				// and no references so this creates a deep copy of struct Event
				*e = *event
				events = append(events, e)
				// delete the advert from adverts
				delete(adverts, key)
			}

			// advertise events to subscribers
			if len(events) > 0 {
				if logger.V(logger.DebugLevel, logger.DefaultLogger) {
					logger.Debugf("Router publishing %d events", len(events))
				}
				go r.publishAdvert(RouteUpdate, events)
			}
		case e := <-r.eventChan:
			// if event is nil, continue
			if e == nil {
				continue
			}

			// If we're not advertising any events then skip processing them entirely
			if r.options.Advertise == AdvertiseNone {
				continue
			}

			// if we only advertise local routes skip processing anything not link local
			if r.options.Advertise == AdvertiseLocal && e.Route.Link != "local" {
				continue
			}

			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("Router processing table event %s for service %s %s", e.Type, e.Route.Service, e.Route.Address)
			}

			// check if we have already registered the route
			hash := e.Route.Hash()
			ev, ok := adverts[hash]
			if !ok {
				ev = e
				adverts[hash] = e
				continue
			}

			// override the route event only if the previous event was different
			if ev.Type != e.Type {
				ev = e
			}
		case <-r.exit:
			if w != nil {
				w.Stop()
			}
			return nil
		}
	}
}

// drain all the events, only called on Stop
func (r *router) drain() {
	for {
		select {
		case <-r.eventChan:
		default:
			return
		}
	}
}

// Advertise stars advertising the routes to the network and returns the advertisements channel to consume from.
// If the router is already advertising it returns the channel to consume from.
// It returns error if the routing table fails to list the routes to advertise.
func (r *router) Advertise() (<-chan *Advert, error) {
	r.Lock()
	defer r.Unlock()

	// already advertising
	if r.eventChan != nil {
		advertChan := make(chan *Advert, 128)
		r.subscribers[uuid.New().String()] = advertChan
		return advertChan, nil
	}

	// list all the routes and pack them into even slice to advertise
	events, err := r.flushRouteEvents(Create)
	if err != nil {
		return nil, fmt.Errorf("failed to flush routes: %s", err)
	}

	// create event channels
	r.eventChan = make(chan *Event)

	// create advert channel
	advertChan := make(chan *Advert, 128)
	r.subscribers[uuid.New().String()] = advertChan

	// advertise your presence
	go r.publishAdvert(Announce, events)

	go func() {
		select {
		case <-r.exit:
			return
		default:
			if err := r.advertiseEvents(); err != nil {
				if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
					logger.Errorf("Error adveritising events: %v", err)
				}
			}
		}
	}()

	return advertChan, nil

}

// Process updates the routing table using the advertised values
func (r *router) Process(a *Advert) error {
	// NOTE: event sorting might not be necessary
	// copy update events intp new slices
	events := make([]*Event, len(a.Events))
	copy(events, a.Events)
	// sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("Router %s processing advert from: %s", r.options.Id, a.Id)
	}

	for _, event := range events {
		// skip if the router is the origin of this route
		if event.Route.Router == r.options.Id {
			if logger.V(logger.TraceLevel, logger.DefaultLogger) {
				logger.Tracef("Router skipping processing its own route: %s", r.options.Id)
			}
			continue
		}
		// create a copy of the route
		route := event.Route
		action := event.Type

		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("Router %s applying %s from router %s for service %s %s", r.options.Id, action, route.Router, route.Service, route.Address)
		}

		if err := r.manageRoute(route, action.String()); err != nil {
			return fmt.Errorf("failed applying action %s to routing table: %s", action, err)
		}
	}

	return nil
}

// flushRouteEvents returns a slice of events, one per each route in the routing table
func (r *router) flushRouteEvents(evType EventType) ([]*Event, error) {
	// get a list of routes for each service in our routing table
	// for the configured advertising strategy
	q := []QueryOption{
		QueryStrategy(r.options.Advertise),
	}

	routes, err := r.Table().Query(q...)
	if err != nil && err != ErrRouteNotFound {
		return nil, err
	}

	if logger.V(logger.DebugLevel, logger.DefaultLogger) {
		logger.Debugf("Router advertising %d routes with strategy %s", len(routes), r.options.Advertise)
	}

	// build a list of events to advertise
	events := make([]*Event, len(routes))
	var i int

	for _, route := range routes {
		event := &Event{
			Type:      evType,
			Timestamp: time.Now(),
			Route:     route,
		}
		events[i] = event
		i++
	}

	return events, nil
}

// Lookup routes in the routing table
func (r *router) Lookup(q ...QueryOption) ([]Route, error) {
	opts := NewQuery(q...)

	// setup the network if we're not already watching it
	if err := r.watchNetwork(opts.Network); err != nil {
		return nil, err
	}

	return r.table.Query(q...)
}

// Watch routes
func (r *router) Watch(opts ...WatchOption) (Watcher, error) {
	return r.table.Watch(opts...)
}

// Close the router
func (r *router) Close() error {
	r.Lock()
	defer r.Unlock()

	select {
	case <-r.exit:
		return nil
	default:
		close(r.exit)

		// extract the events
		r.drain()

		r.sub.Lock()
		// close advert subscribers
		for id, sub := range r.subscribers {
			// close the channel
			close(sub)
			// delete the subscriber
			delete(r.subscribers, id)
		}
		r.sub.Unlock()
	}

	// remove event chan
	r.eventChan = nil

	return nil
}

// String prints debugging information about router
func (r *router) String() string {
	return "registry"
}

// watchNetwork starts watching a network if it isn't already being watched
func (r *router) watchNetwork(network string) error {
	if len(network) == 0 {
		return nil
	}

	r.RLock()
	_, ok := r.networks[network]
	r.RUnlock()
	if ok {
		return nil
	}

	r.Lock()
	defer r.Unlock()

	r.resetWatchers[network] = make(chan bool)
	go r.watchRegistry(network)
	return r.populateRoutes(r.options.Registry, network)
}
