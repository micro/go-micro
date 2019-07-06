package router

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-log"
	"github.com/micro/go-micro/registry"
	"github.com/olekukonko/tablewriter"
)

const (
	// UpdateRoutePenalty penalises route updates
	UpdateRoutePenalty = 500
	// DeleteRoutePenalty penalises route deletes
	DeleteRoutePenalty = 1000
	// AdvertiseTick is time interval in which we advertise route updates
	AdvertiseTick = 5 * time.Second
	// AdvertSuppress is advert suppression threshold
	AdvertSuppress = 2000
	// AdvertRecover is advert suppression recovery threshold
	AdvertRecover = 750
	// PenaltyDecay is the "half-life" of the penalty
	PenaltyDecay = 1.15
)

// router provides default router implementation
type router struct {
	opts       Options
	status     Status
	exit       chan struct{}
	eventChan  chan *Event
	advertChan chan *Advert
	wg         *sync.WaitGroup
	sync.RWMutex
}

// newRouter creates new router and returns it
func newRouter(opts ...Option) Router {
	// get default options
	options := DefaultOptions()

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	return &router{
		opts:       options,
		status:     Status{Error: nil, Code: Init},
		exit:       make(chan struct{}),
		eventChan:  make(chan *Event),
		advertChan: make(chan *Advert),
		wg:         &sync.WaitGroup{},
	}
}

// Init initializes router with given options
func (r *router) Init(opts ...Option) error {
	for _, o := range opts {
		o(&r.opts)
	}
	return nil
}

// Options returns router options
func (r *router) Options() Options {
	return r.opts
}

// ID returns router ID
func (r *router) ID() string {
	return r.opts.ID
}

// Table returns routing table
func (r *router) Table() Table {
	return r.opts.Table
}

// Address returns router's bind address
func (r *router) Address() string {
	return r.opts.Address
}

// Network returns the address router advertises to the network
func (r *router) Network() string {
	return r.opts.Network
}

// addServiceRoutes adds all services in given registry to the routing table.
// NOTE: this is a one-off operation done when bootstrapping the router
// It returns error if either the services failed to be listed or
// if any of the the routes failed to be added to the routing table.
func (r *router) addServiceRoutes(reg registry.Registry, network string, metric int) error {
	services, err := reg.ListServices()
	if err != nil {
		return fmt.Errorf("failed listing services: %v", err)
	}

	// add each service node as a separate route
	for _, service := range services {
		// get the service to retrieve all its info
		srvs, err := reg.GetService(service.Name)
		if err != nil {
			log.Logf("r.addServiceRoutes() GetService() error: %v", err)
			continue
		}

		// create a flat slide of nodes
		var nodes []*registry.Node
		for _, s := range srvs {
			nodes = append(nodes, s.Nodes...)
		}

		// range over the flat slice of nodes
		for _, node := range nodes {
			route := Route{
				Destination: service.Name,
				Gateway:     node.Address,
				Router:      r.opts.Address,
				Network:     r.opts.Network,
				Metric:      metric,
			}
			if err := r.opts.Table.Add(route); err != nil && err != ErrDuplicateRoute {
				return fmt.Errorf("error adding route for service %s: %s", service.Name, err)
			}
		}
	}

	return nil
}

// watchServices watches services in given registry and updates the routing table accordingly.
// It returns error if the service registry watcher stops or if the routing table can't be updated.
func (r *router) watchServices(w registry.Watcher) error {
	// wait in the background for the router to stop
	// when the router stops, stop the watcher and exit
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		<-r.exit
		w.Stop()
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

		log.Logf("r.watchServices() new service event: Action: %s Service: %v", res.Action, res.Service)

		switch res.Action {
		case "create":
			// range over the flat slice of nodes
			for _, node := range res.Service.Nodes {
				gateway := node.Address
				if node.Port > 0 {
					gateway = fmt.Sprintf("%s:%d", node.Address, node.Port)
				}
				route := Route{
					Destination: res.Service.Name,
					Gateway:     gateway,
					Router:      r.opts.Address,
					Network:     r.opts.Network,
					Metric:      DefaultLocalMetric,
				}
				if err := r.opts.Table.Add(route); err != nil && err != ErrDuplicateRoute {
					return fmt.Errorf("error adding route for service %s: %s", res.Service.Name, err)
				}
			}
		case "delete":
			for _, node := range res.Service.Nodes {
				route := Route{
					Destination: res.Service.Name,
					Gateway:     node.Address,
					Router:      r.opts.Address,
					Network:     r.opts.Network,
					Metric:      DefaultLocalMetric,
				}
				// only return error if the route is not in the table, but something else has failed
				if err := r.opts.Table.Delete(route); err != nil && err != ErrRouteNotFound {
					return fmt.Errorf("failed adding route for service %v: %s", res.Service.Name, err)
				}
			}
		}
	}

	return watchErr
}

// watchTable watches routing table entries and either adds or deletes locally registered service to/from network registry
// It returns error if the locally registered services either fails to be added/deleted to/from network registry.
func (r *router) watchTable(w Watcher) error {
	// wait in the background for the router to stop
	// when the router stops, stop the watcher and exit
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		<-r.exit
		w.Stop()
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

func eventFlap(curr, prev *Event) bool {
	if curr.Type == UpdateEvent && prev.Type == UpdateEvent {
		// update flap: this can be either metric or whatnot
		log.Logf("eventFlap(): Update flap")
		return true
	}

	if curr.Type == CreateEvent && prev.Type == DeleteEvent || curr.Type == DeleteEvent && prev.Type == CreateEvent {
		log.Logf("eventFlap(): Create/Delete flap")
		return true
	}

	return false
}

// processEvents processes routing table events.
// It suppresses unhealthy flapping events and advertises healthy events upstream.
func (r *router) processEvents() error {
	// ticker to periodically scan event for advertising
	ticker := time.NewTicker(AdvertiseTick)

	// advertEvent is a table event enriched with advert data
	type advertEvent struct {
		*Event
		timestamp    time.Time
		penalty      float64
		isSuppressed bool
		isFlapping   bool
	}

	// eventMap is a map of advert events that might end up being advertised
	eventMap := make(map[uint64]*advertEvent)
	// lock to protect access to eventMap
	mu := &sync.RWMutex{}
	// waitgroup to manage advertisement goroutines
	var wg sync.WaitGroup

process:
	for {
		select {
		case <-ticker.C:
			var events []*Event
			// decay the penalties of existing events
			mu.Lock()
			for advert, event := range eventMap {
				delta := time.Since(event.timestamp).Seconds()
				event.penalty = event.penalty * math.Exp(delta)
				// suppress or recover the event based on its current penalty
				if !event.isSuppressed && event.penalty > AdvertSuppress {
					event.isSuppressed = true
				} else if event.penalty < AdvertRecover {
					event.isSuppressed = false
					event.isFlapping = false
				}
				if !event.isFlapping {
					e := new(Event)
					*e = *event.Event
					events = append(events, e)
					// this deletes the advertised event from the map
					delete(eventMap, advert)
				}
			}
			mu.Unlock()

			if len(events) > 0 {
				wg.Add(1)
				go func(events []*Event) {
					defer wg.Done()

					log.Logf("go advertise(): start")

					a := &Advert{
						ID:        r.ID(),
						Timestamp: time.Now(),
						Events:    events,
					}

					select {
					case r.advertChan <- a:
						mu.Lock()
						// once we've advertised the events, we need to delete them
						for _, event := range a.Events {
							delete(eventMap, event.Route.Hash())
						}
						mu.Unlock()
					case <-r.exit:
						log.Logf("go advertise(): exit")
						return
					}
					log.Logf("go advertise(): exit")
				}(events)
			}
		case e := <-r.eventChan:
			// if event is nil, break
			if e == nil {
				continue
			}
			log.Logf("r.processEvents(): event received:\n%s", e)
			// determine the event penalty
			var penalty float64
			switch e.Type {
			case UpdateEvent:
				penalty = UpdateRoutePenalty
			case CreateEvent, DeleteEvent:
				penalty = DeleteRoutePenalty
			}
			// we use route hash as eventMap key
			hash := e.Route.Hash()
			event, ok := eventMap[hash]
			if !ok {
				event = &advertEvent{
					Event:     e,
					penalty:   penalty,
					timestamp: time.Now(),
				}
				eventMap[hash] = event
				continue
			}
			// update penalty for existing event: decay existing and add new penalty
			delta := time.Since(event.timestamp).Seconds()
			event.penalty = event.penalty*math.Exp(delta) + penalty
			event.timestamp = time.Now()
			// suppress or recover the event based on its current penalty
			if !event.isSuppressed && event.penalty > AdvertSuppress {
				event.isSuppressed = true
			} else if event.penalty < AdvertRecover {
				event.isSuppressed = false
			}
			// if not suppressed decide if if its flapping
			if !event.isSuppressed {
				// detect if its flapping
				event.isFlapping = eventFlap(e, event.Event)
			}
		case <-r.exit:
			break process
		}
	}

	// first wait for the advertiser to finish
	wg.Wait()
	// close the advert channel
	close(r.advertChan)

	log.Logf("r.processEvents(): event processor stopped")

	return nil
}

// manage watches router errors and takes appropriate actions
func (r *router) manage(errChan <-chan error) {
	defer r.wg.Done()

	log.Logf("r.manage(): manage start")

	var code StatusCode
	var err error

	select {
	case <-r.exit:
		code = Stopped
	case err = <-errChan:
		code = Error
	}

	log.Logf("r.manage(): manage exiting")

	r.Lock()
	defer r.Unlock()
	status := Status{
		Code:  code,
		Error: err,
	}
	r.status = status

	// stop the router if some error happened
	if err != nil && code != Stopped {
		// this will stop watchers which will close r.advertChan
		close(r.exit)
		// drain the advertise channel
		for range r.advertChan {
		}
		// drain the event channel
		for range r.eventChan {
		}
	}

	log.Logf("r.manage(): manage exit")
}

// Advertise advertises the routes to the network.
// It returns error if any of the launched goroutines fail with error.
func (r *router) Advertise() (<-chan *Advert, error) {
	r.Lock()
	defer r.Unlock()

	if r.status.Code != Running {
		// add local service routes into the routing table
		if err := r.addServiceRoutes(r.opts.Registry, "local", DefaultLocalMetric); err != nil {
			return nil, fmt.Errorf("failed adding routes: %v", err)
		}
		log.Logf("Routing table:\n%s", r.opts.Table)
		// add default gateway into routing table
		if r.opts.Gateway != "" {
			// note, the only non-default value is the gateway
			route := Route{
				Destination: "*",
				Gateway:     r.opts.Gateway,
				Router:      "*",
				Network:     "*",
				Metric:      DefaultLocalMetric,
			}
			if err := r.opts.Table.Add(route); err != nil {
				return nil, fmt.Errorf("failed adding default gateway route: %s", err)
			}
		}

		// NOTE: we only need to recreate the exit/advertChan if the router errored or was stopped
		// TODO: these channels most likely won't have to be the struct fields
		if r.status.Code == Error || r.status.Code == Stopped {
			r.exit = make(chan struct{})
			r.eventChan = make(chan *Event)
			r.advertChan = make(chan *Advert)
		}

		// routing table watcher which watches all routes i.e. to every destination
		tableWatcher, err := r.opts.Table.Watch(WatchDestination("*"))
		if err != nil {
			return nil, fmt.Errorf("failed creating routing table watcher: %v", err)
		}
		// service registry watcher
		svcWatcher, err := r.opts.Registry.Watch()
		if err != nil {
			return nil, fmt.Errorf("failed creating service registry watcher: %v", err)
		}

		// error channel collecting goroutine errors
		errChan := make(chan error, 3)

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			log.Logf("r.Advertise(): r.watchServices() start")
			// watch local registry and register routes in routine table
			errChan <- r.watchServices(svcWatcher)
			log.Logf("r.Advertise(): r.watchServices() exit")
		}()

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			log.Logf("r.Advertise(): r.watchTable() start")
			// watch local registry and register routes in routing table
			errChan <- r.watchTable(tableWatcher)
			log.Logf("r.Advertise(): r.watchTable() exit")
		}()

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			log.Logf("r.Advertise(): r.processEvents() start")
			// listen to routing table events and process them
			errChan <- r.processEvents()
			log.Logf("r.Advertise(): r.processEvents() exit")
		}()

		r.wg.Add(1)
		go r.manage(errChan)

		// mark router as running and set its Error to nil
		status := Status{
			Code:  Running,
			Error: nil,
		}
		r.status = status
	}

	return r.advertChan, nil
}

// Update updates the routing table using the advertised values
func (r *router) Update(a *Advert) error {
	// NOTE: event sorting might not be necessary
	// copy update events intp new slices
	events := make([]*Event, len(a.Events))
	copy(events, a.Events)
	// sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	for _, event := range events {
		// we extract the route from advertisement and update the routing table
		route := Route{
			Destination: event.Route.Destination,
			Gateway:     event.Route.Gateway,
			Router:      event.Route.Router,
			Network:     event.Route.Network,
			Metric:      event.Route.Metric,
			Policy:      AddIfNotExists,
		}
		if err := r.opts.Table.Update(route); err != nil {
			return fmt.Errorf("failed updating routing table: %v", err)
		}
	}

	return nil
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
	log.Logf("r.Stop(): Stopping router")
	r.RLock()
	// only close the channel if the router is running
	if r.status.Code == Running {
		// notify all goroutines to finish
		close(r.exit)
		log.Logf("r.Stop(): exit closed")
		// drain the advertise channel
		for range r.advertChan {
		}
		log.Logf("r.Stop(): advert channel drained")
		// drain the event channel
		for range r.eventChan {
		}
		log.Logf("r.Stop(): event channel drained")
	}
	r.RUnlock()

	// wait for all goroutines to finish
	r.wg.Wait()

	log.Logf("r.Stop(): Router stopped")
	return nil
}

// String prints debugging information about router
func (r *router) String() string {
	sb := &strings.Builder{}

	table := tablewriter.NewWriter(sb)
	table.SetHeader([]string{"ID", "Address", "Network", "Table", "Status"})

	data := []string{
		r.opts.ID,
		r.opts.Address,
		r.opts.Network,
		fmt.Sprintf("%d", r.opts.Table.Size()),
		r.status.Code.String(),
	}
	table.Append(data)

	// render table into sb
	table.Render()

	return sb.String()
}
