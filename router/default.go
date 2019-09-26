package router

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/util/log"
)

const (
	// AdvertiseEventsTick is time interval in which the router advertises route updates
	AdvertiseEventsTick = 5 * time.Second
	// AdvertiseTableTick is time interval in which router advertises all routes found in routing table
	AdvertiseTableTick = 1 * time.Minute
	// AdvertiseFlushTick is time the yet unconsumed advertisements are flush i.e. discarded
	AdvertiseFlushTick = 15 * time.Second
	// AdvertSuppress is advert suppression threshold
	AdvertSuppress = 2000.0
	// AdvertRecover is advert recovery threshold
	AdvertRecover = 750.0
	// DefaultAdvertTTL is default advertisement TTL
	DefaultAdvertTTL = 1 * time.Minute
	// DeletePenalty penalises route deletion
	DeletePenalty = 1000.0
	// UpdatePenalty penalises route updates
	UpdatePenalty = 500.0
	// PenaltyHalfLife is the time the advert penalty decays to half its value
	PenaltyHalfLife = 2.0
	// MaxSuppressTime defines time after which the suppressed advert is deleted
	MaxSuppressTime = 5 * time.Minute
)

var (
	// PenaltyDecay is a coefficient which controls the speed the advert penalty decays
	PenaltyDecay = math.Log(2) / PenaltyHalfLife
)

// router implements default router
type router struct {
	sync.RWMutex
	options   Options
	status    Status
	table     *table
	exit      chan struct{}
	errChan   chan error
	eventChan chan *Event
	advertWg  *sync.WaitGroup
	wg        *sync.WaitGroup

	// advert subscribers
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

	// set initial status to Stopped
	status := Status{Code: Stopped, Error: nil}

	return &router{
		options:     options,
		status:      status,
		table:       newTable(),
		advertWg:    &sync.WaitGroup{},
		wg:          &sync.WaitGroup{},
		subscribers: make(map[string]chan *Advert),
	}
}

// Init initializes router with given options
func (r *router) Init(opts ...Option) error {
	r.Lock()
	defer r.Unlock()

	for _, o := range opts {
		o(&r.options)
	}

	return nil
}

// Options returns router options
func (r *router) Options() Options {
	r.Lock()
	options := r.options
	r.Unlock()

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
	case "solicit":
		// nothing to do here
		return nil
	default:
		return fmt.Errorf("failed to manage route for service %s: unknown action %s", route.Service, action)
	}

	return nil
}

// manageServiceRoutes applies action to all routes of the service.
// It returns error of the action fails with error.
func (r *router) manageServiceRoutes(service *registry.Service, action string) error {
	// action is the routing table action
	action = strings.ToLower(action)

	// take route action on each service node
	for _, node := range service.Nodes {
		route := Route{
			Service: service.Name,
			Address: node.Address,
			Gateway: "",
			Network: r.options.Network,
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

// manageRegistryRoutes applies action to all routes of each service found in the registry.
// It returns error if either the services failed to be listed or the routing table action fails.
func (r *router) manageRegistryRoutes(reg registry.Registry, action string) error {
	services, err := reg.ListServices()
	if err != nil {
		return fmt.Errorf("failed listing services: %v", err)
	}

	// add each service node as a separate route
	for _, service := range services {
		// get the service to retrieve all its info
		srvs, err := reg.GetService(service.Name)
		if err != nil {
			continue
		}
		// manage the routes for all returned services
		for _, srv := range srvs {
			if err := r.manageServiceRoutes(srv, action); err != nil {
				return err
			}
		}
	}

	return nil
}

// watchRegistry watches registry and updates routing table based on the received events.
// It returns error if either the registry watcher fails with error or if the routing table update fails.
func (r *router) watchRegistry(w registry.Watcher) error {
	exit := make(chan bool)

	defer func() {
		// close the exit channel when the go routine finishes
		close(exit)
		r.wg.Done()
	}()

	// wait in the background for the router to stop
	// when the router stops, stop the watcher and exit
	r.wg.Add(1)
	go func() {
		defer w.Stop()

		select {
		case <-r.exit:
			return
		case <-exit:
			return
		}
	}()

	var watchErr error

	for {
		res, err := w.Next()
		if err != nil {
			if err != registry.ErrWatcherStopped {
				watchErr = err
			}
			break
		}

		if err := r.manageServiceRoutes(res.Service, res.Action); err != nil {
			return err
		}
	}

	return watchErr
}

// watchTable watches routing table entries and either adds or deletes locally registered service to/from network registry
// It returns error if the locally registered services either fails to be added/deleted to/from network registry.
func (r *router) watchTable(w Watcher) error {
	exit := make(chan bool)

	defer func() {
		// close the exit channel when the go routine finishes
		close(exit)
		r.wg.Done()
	}()

	// wait in the background for the router to stop
	// when the router stops, stop the watcher and exit
	r.wg.Add(1)
	go func() {
		defer w.Stop()

		select {
		case <-r.exit:
			return
		case <-exit:
			return
		}
	}()

	var watchErr error

	for {
		event, err := w.Next()
		if err != nil {
			if err != ErrWatcherStopped {
				watchErr = err
			}
			break
		}

		select {
		case <-r.exit:
			close(r.eventChan)
			return nil
		case r.eventChan <- event:
		}
	}

	// close event channel on error
	close(r.eventChan)

	return watchErr
}

// publishAdvert publishes router advert to advert channel
// NOTE: this might cease to be a dedicated method in the future
func (r *router) publishAdvert(advType AdvertType, events []*Event) {
	defer r.advertWg.Done()

	a := &Advert{
		Id:        r.options.Id,
		Type:      advType,
		TTL:       DefaultAdvertTTL,
		Timestamp: time.Now(),
		Events:    events,
	}

	log.Debugf("Router publishing advert; %+v", a)
	r.RLock()
	for _, sub := range r.subscribers {
		// check the exit chan first
		select {
		case <-r.exit:
			r.RUnlock()
			return
		default:
		}

		// now send the message
		select {
		case sub <- a:
		default:
		}
	}
	r.RUnlock()
}

// advertiseTable advertises the whole routing table to the network
func (r *router) advertiseTable() error {
	// create table advertisement ticker
	ticker := time.NewTicker(AdvertiseTableTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// list routing table routes to announce
			routes, err := r.table.List()
			if err != nil {
				return fmt.Errorf("failed listing routes: %s", err)
			}
			// collect all the added routes before we attempt to add default gateway
			events := make([]*Event, len(routes))
			for i, route := range routes {
				event := &Event{
					Type:      Update,
					Timestamp: time.Now(),
					Route:     route,
				}
				events[i] = event
			}

			// advertise all routes as Update events to subscribers
			if len(events) > 0 {
				log.Debugf("Router flushing table with %d events: %s", len(events), r.options.Id)
				r.advertWg.Add(1)
				go r.publishAdvert(RouteUpdate, events)
			}
		case <-r.exit:
			return nil
		}
	}
}

// routeAdvert contains a route event to be advertised
type routeAdvert struct {
	// event received from routing table
	event *Event
	// lastUpdate records the time of the last advert update
	lastUpdate time.Time
	// penalty is current advert penalty
	penalty float64
	// isSuppressed flags the advert suppression
	isSuppressed bool
	// suppressTime records the time interval the advert has been suppressed for
	suppressTime time.Time
}

// advertiseEvents advertises routing table events
// It suppresses unhealthy flapping events and advertises healthy events upstream.
func (r *router) advertiseEvents() error {
	// ticker to periodically scan event for advertising
	ticker := time.NewTicker(AdvertiseEventsTick)
	defer ticker.Stop()

	// advertMap is a map of advert events
	advertMap := make(map[uint64]*routeAdvert)

	// routing table watcher
	tableWatcher, err := r.Watch()
	if err != nil {
		return fmt.Errorf("failed creating routing table watcher: %v", err)
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		select {
		case r.errChan <- r.watchTable(tableWatcher):
		case <-r.exit:
		}
	}()

	for {
		select {
		case <-ticker.C:
			var events []*Event
			// collect all events which are not flapping
			for key, advert := range advertMap {
				// decay the event penalty
				delta := time.Since(advert.lastUpdate).Seconds()
				advert.penalty = advert.penalty * math.Exp(-delta*PenaltyDecay)

				// suppress/recover the event based on its penalty level
				switch {
				case advert.penalty > AdvertSuppress && !advert.isSuppressed:
					log.Debugf("Router supressing advert %d for route %s", key, advert.event.Route.Address)
					advert.isSuppressed = true
					advert.suppressTime = time.Now()
				case advert.penalty < AdvertRecover && advert.isSuppressed:
					log.Debugf("Router recovering advert %d for route %s", key, advert.event.Route.Address)
					advert.isSuppressed = false
				}

				// max suppression time threshold has been reached, delete the advert
				if advert.isSuppressed {
					if time.Since(advert.suppressTime) > MaxSuppressTime {
						delete(advertMap, key)
						continue
					}
				}

				if !advert.isSuppressed {
					e := new(Event)
					*e = *(advert.event)
					events = append(events, e)
					// delete the advert from the advertMap
					delete(advertMap, key)
				}
			}

			// advertise all Update events to subscribers
			if len(events) > 0 {
				r.advertWg.Add(1)
				log.Debugf("Router publishing %d events", len(events))
				go r.publishAdvert(RouteUpdate, events)
			}
		case e := <-r.eventChan:
			// if event is nil, continue
			if e == nil {
				continue
			}
			log.Debugf("Router processing table event %s for service %s", e.Type, e.Route.Address)
			// determine the event penalty
			var penalty float64
			switch e.Type {
			case Update:
				penalty = UpdatePenalty
			case Delete:
				penalty = DeletePenalty
			}

			// check if we have already registered the route
			hash := e.Route.Hash()
			advert, ok := advertMap[hash]
			if !ok {
				advert = &routeAdvert{
					event:      e,
					penalty:    penalty,
					lastUpdate: time.Now(),
				}
				advertMap[hash] = advert
				continue
			}

			// override the route event only if the last event was different
			if advert.event.Type != e.Type {
				advert.event = e
			}

			// update event penalty and timestamp
			advert.lastUpdate = time.Now()
			advert.penalty += penalty
			log.Debugf("Router advert %d for route %s event penalty: %f", hash, advert.event.Route.Address, advert.penalty)
		case <-r.exit:
			// first wait for the advertiser to finish
			r.advertWg.Wait()
			return nil
		}
	}
}

// close closes exit channels
func (r *router) close() {
	// notify all goroutines to finish
	close(r.exit)

	// drain the advertise channel only if advertising
	if r.status.Code == Advertising {
		// drain the event channel
		for range r.eventChan {
		}

		// close advert subscribers
		for id, sub := range r.subscribers {
			// close the channel
			close(sub)

			// delete the subscriber
			delete(r.subscribers, id)
		}
	}

	// mark the router as Stopped and set its Error to nil
	r.status = Status{Code: Stopped, Error: nil}
}

// watchErrors watches router errors and takes appropriate actions
func (r *router) watchErrors() {
	var err error

	select {
	case <-r.exit:
		return
	case err = <-r.errChan:
	}

	r.Lock()
	defer r.Unlock()
	// if the router is not stopped, stop it
	if r.status.Code != Stopped {
		// close all the channels
		r.close()
		// set the status error
		if err != nil {
			r.status.Error = err
		}
	}
}

// Start starts the router
func (r *router) Start() error {
	r.Lock()
	defer r.Unlock()

	// only start if we're stopped
	if r.status.Code != Stopped {
		return nil
	}

	// add all local service routes into the routing table
	if err := r.manageRegistryRoutes(r.options.Registry, "create"); err != nil {
		e := fmt.Errorf("failed adding registry routes: %s", err)
		r.status = Status{Code: Error, Error: e}
		return e
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
			e := fmt.Errorf("failed adding default gateway route: %s", err)
			r.status = Status{Code: Error, Error: e}
			return e
		}
	}

	// create error and exit channels
	r.errChan = make(chan error, 1)
	r.exit = make(chan struct{})

	// registry watcher
	regWatcher, err := r.options.Registry.Watch()
	if err != nil {
		e := fmt.Errorf("failed creating registry watcher: %v", err)
		r.status = Status{Code: Error, Error: e}
		return e
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		select {
		case r.errChan <- r.watchRegistry(regWatcher):
		case <-r.exit:
		}
	}()

	// watch for errors and cleanup
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.watchErrors()
	}()

	// mark router as Running
	r.status = Status{Code: Running, Error: nil}

	return nil
}

// Advertise stars advertising the routes to the network and returns the advertisements channel to consume from.
// If the router is already advertising it returns the channel to consume from.
// It returns error if either the router is not running or if the routing table fails to list the routes to advertise.
func (r *router) Advertise() (<-chan *Advert, error) {
	r.Lock()
	defer r.Unlock()

	switch r.status.Code {
	case Advertising:
		advertChan := make(chan *Advert, 128)
		r.subscribers[uuid.New().String()] = advertChan
		return advertChan, nil
	case Running:
		// list all the routes and pack them into even slice to advertise
		events, err := r.flushRouteEvents(Create)
		if err != nil {
			return nil, fmt.Errorf("failed to flush routes: %s", err)
		}

		// create event channels
		r.eventChan = make(chan *Event)

		// advertise your presence
		r.advertWg.Add(1)
		go r.publishAdvert(Announce, events)

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			select {
			case r.errChan <- r.advertiseEvents():
			case <-r.exit:
			}
		}()

		r.advertWg.Add(1)
		go func() {
			defer r.advertWg.Done()
			// advertise the whole routing table
			select {
			case r.errChan <- r.advertiseTable():
			case <-r.exit:
			}
		}()

		// mark router as Running and set its Error to nil
		r.status = Status{Code: Advertising, Error: nil}

		// create advert channel
		advertChan := make(chan *Advert, 128)
		r.subscribers[uuid.New().String()] = advertChan

		return advertChan, nil
	case Stopped:
		return nil, fmt.Errorf("not running")
	}

	return nil, fmt.Errorf("error: %s", r.status.Error)
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

	log.Debugf("Router %s processing advert from: %s", r.options.Id, a.Id)

	for _, event := range events {
		// skip if the router is the origin of this route
		if event.Route.Router == r.options.Id {
			log.Debugf("Router skipping processing its own route: %s", r.options.Id)
			continue
		}
		// create a copy of the route
		route := event.Route
		action := event.Type
		log.Debugf("Router %s applying %s from router %s for address: %s", r.options.Id, action, route.Router, route.Address)
		if err := r.manageRoute(route, fmt.Sprintf("%s", action)); err != nil {
			return fmt.Errorf("failed applying action %s to routing table: %s", action, err)
		}
	}

	return nil
}

// flushRouteEvents returns a slice of events, one per each route in the routing table
func (r *router) flushRouteEvents(evType EventType) ([]*Event, error) {
	// list all routes
	routes, err := r.table.List()
	if err != nil {
		return nil, fmt.Errorf("failed listing routes: %s", err)
	}

	// build a list of events to advertise
	events := make([]*Event, len(routes))
	for i, route := range routes {
		event := &Event{
			Type:      evType,
			Timestamp: time.Now(),
			Route:     route,
		}
		events[i] = event
	}

	return events, nil
}

// Solicit advertises all of its routes to the network
// It returns error if the router fails to list the routes
func (r *router) Solicit() error {
	events, err := r.flushRouteEvents(Update)
	if err != nil {
		return fmt.Errorf("failed solicit routes: %s", err)
	}

	// advertise the routes
	r.advertWg.Add(1)
	go r.publishAdvert(RouteUpdate, events)

	return nil
}

// Lookup routes in the routing table
func (r *router) Lookup(q Query) ([]Route, error) {
	return r.table.Query(q)
}

// Watch routes
func (r *router) Watch(opts ...WatchOption) (Watcher, error) {
	return r.table.Watch(opts...)
}

// Status returns router status
func (r *router) Status() Status {
	r.RLock()
	defer r.RUnlock()

	// make a copy of the status
	status := r.status

	return status
}

// Stop stops the router
func (r *router) Stop() error {
	r.Lock()
	defer r.Unlock()

	switch r.status.Code {
	case Stopped, Error:
		return r.status.Error
	case Running, Advertising:
		// close all the channels
		r.close()
	}

	// wait for all goroutines to finish
	r.wg.Wait()

	return nil
}

// String prints debugging information about router
func (r *router) String() string {
	return "memory"
}
