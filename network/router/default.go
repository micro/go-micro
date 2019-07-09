package router

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-log"
	"github.com/micro/go-micro/network/router/table"
	"github.com/micro/go-micro/registry"
	"github.com/olekukonko/tablewriter"
)

const (
	// AdvertiseTick is time interval in which we advertise route updates
	AdvertiseTick = 5 * time.Second
	// AdvertSuppress is advert suppression threshold
	AdvertSuppress = 2000
	// AdvertRecover is advert recovery threshold
	AdvertRecover = 750
	// DefaultAdvertTTL is default advertisement TTL
	DefaultAdvertTTL = time.Minute
	// PenaltyDecay is the penalty decay
	PenaltyDecay = 1.15
	// Delete penalises route addition and deletion
	Delete = 1000
	// UpdatePenalty penalises route updates
	UpdatePenalty = 500
)

// router provides default router implementation
type router struct {
	opts       Options
	status     Status
	exit       chan struct{}
	eventChan  chan *table.Event
	advertChan chan *Advert
	advertWg   *sync.WaitGroup
	wg         *sync.WaitGroup
	sync.RWMutex
}

// newRouter creates a new router and returns it
func newRouter(opts ...Option) Router {
	// get default options
	options := DefaultOptions()

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	return &router{
		opts:       options,
		status:     Status{Error: nil, Code: Stopped},
		exit:       make(chan struct{}),
		eventChan:  make(chan *table.Event),
		advertChan: make(chan *Advert),
		advertWg:   &sync.WaitGroup{},
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
func (r *router) Table() table.Table {
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

// manageServiceRoutes manages routes for a given service.
// It returns error of the routing table action fails.
func (r *router) manageServiceRoutes(service *registry.Service, action string) error {
	// action is the routing table action
	action = strings.ToLower(action)
	// take route action on each service node
	for _, node := range service.Nodes {
		route := table.Route{
			Service: service.Name,
			Address: node.Address,
			Gateway: "",
			Network: r.opts.Network,
			Link:    table.DefaultLink,
			Metric:  table.DefaultLocalMetric,
		}
		switch action {
		case "insert", "create":
			if err := r.opts.Table.Add(route); err != nil && err != table.ErrDuplicateRoute {
				return fmt.Errorf("failed adding route for service %s: %s", service.Name, err)
			}
		case "delete":
			if err := r.opts.Table.Delete(route); err != nil && err != table.ErrRouteNotFound {
				return fmt.Errorf("failed deleting route for service %v: %s", service.Name, err)
			}
		default:
			return fmt.Errorf("failed to manage route for service %v. Unknown action: %s", service.Name, action)
		}
	}
	return nil
}

// manageRegistryRoutes manages routes for each service found in the registry.
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
			log.Logf("r.manageRegistryRoutes() GetService() error: %v", err)
			continue
		}
		// manage the routes for all returned services
		for _, s := range srvs {
			if err := r.manageServiceRoutes(s, action); err != nil {
				return err
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

		if err := r.manageServiceRoutes(res.Service, res.Action); err != nil {
			return err
		}
	}

	return watchErr
}

// watchTable watches routing table entries and either adds or deletes locally registered service to/from network registry
// It returns error if the locally registered services either fails to be added/deleted to/from network registry.
func (r *router) watchTable(w table.Watcher) error {
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
			if err != table.ErrWatcherStopped {
				watchErr = err
			}
			break
		}

		log.Logf("r.watchTable() new table event: %s", event)

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

func (r *router) advertEvents(advType AdvertType, events []*table.Event) {
	defer r.advertWg.Done()

	log.Logf("r.advertEvents(): start event: %s", advType)

	a := &Advert{
		ID:        r.ID(),
		Type:      advType,
		Timestamp: time.Now(),
		Events:    events,
	}

	select {
	case r.advertChan <- a:
		log.Logf("r.advertEvents(): advertised event: %s", advType)
	case <-r.exit:
		log.Logf("r.advertEvents(): DONE exit")
		return
	}

	log.Logf("r.advertEvents(): REGULAR exit")
}

// isFlapping detects if the event is flapping based on the current and previous event status.
func isFlapping(curr, prev *table.Event) bool {
	if curr.Type == table.Update && prev.Type == table.Update {
		log.Logf("isFlapping(): Update flap")
		return true
	}

	if curr.Type == table.Insert && prev.Type == table.Delete || curr.Type == table.Delete && prev.Type == table.Insert {
		log.Logf("isFlapping(): Create/Delete flap")
		return true
	}

	return false
}

// updateEvent is a table event enriched with advertisement data
type updateEvent struct {
	*table.Event
	// timestamp marks the time the event has been received
	timestamp time.Time
	// penalty is current event penalty
	penalty float64
	// isSuppressed flags if the event should be considered for flap detection
	isSuppressed bool
	// isFlapping marks the event as flapping event
	isFlapping bool
}

// processEvents processes routing table events.
// It suppresses unhealthy flapping events and advertises healthy events upstream.
func (r *router) processEvents() error {
	// ticker to periodically scan event for advertising
	ticker := time.NewTicker(AdvertiseTick)
	// eventMap is a map of advert events
	eventMap := make(map[uint64]*updateEvent)

	for {
		select {
		case <-ticker.C:
			var events []*table.Event
			// collect all events which are not flapping
			for key, event := range eventMap {
				if !event.isFlapping && !event.isSuppressed {
					e := new(table.Event)
					*e = *event.Event
					events = append(events, e)
					// this deletes the advertised event from the map
					delete(eventMap, key)
				}
			}

			if len(events) > 0 {
				r.advertWg.Add(1)
				go r.advertEvents(Update, events)
			}
		case e := <-r.eventChan:
			// event timestamp
			now := time.Now()
			// if event is nil, continue
			if e == nil {
				continue
			}
			log.Logf("r.processEvents(): event received:\n%s", e)
			// determine the event penalty
			var penalty float64
			switch e.Type {
			case table.Update:
				penalty = UpdatePenalty
			case table.Delete:
				penalty = Delete
			}
			// we use route hash as eventMap key
			hash := e.Route.Hash()
			event, ok := eventMap[hash]
			if !ok {
				event = &updateEvent{
					Event:     e,
					penalty:   penalty,
					timestamp: time.Now(),
				}
				eventMap[hash] = event
				continue
			}
			// update penalty for existing event: decay existing and add new penalty
			delta := time.Since(event.timestamp).Seconds()
			event.penalty = event.penalty*math.Exp(-delta) + penalty
			event.timestamp = now
			// suppress or recover the event based on its current penalty
			if !event.isSuppressed && event.penalty > AdvertSuppress {
				event.isSuppressed = true
			} else if event.penalty < AdvertRecover {
				event.isSuppressed = false
			}
			// if not suppressed decide if if its flapping
			if !event.isSuppressed {
				// detect if its flapping by comparing current and previous event
				event.isFlapping = isFlapping(e, event.Event)
			}
		case <-r.exit:
			// first wait for the advertiser to finish
			r.advertWg.Wait()
			// close the advert channel
			close(r.advertChan)
			log.Logf("r.processEvents(): event processor stopped")
			return nil
		}
	}

	// we probably never reach this place
	log.Logf("r.processEvents(): event processor stopped")

	return nil
}

// watchErrors watches router errors and takes appropriate actions
func (r *router) watchErrors(errChan <-chan error) {
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

	log.Logf("r.watchErrors(): watchErrors exiting")

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
		log.Logf("r.watchErrors(): advert channel drained")
		// drain the event channel
		for range r.eventChan {
		}
		log.Logf("r.watchErrors(): event channel drained")
	}

	log.Logf("r.watchErrors(): watchErrors exit")
}

// Advertise advertises the routes to the network.
// It returns error if any of the launched goroutines fail with error.
func (r *router) Advertise() (<-chan *Advert, error) {
	r.Lock()
	defer r.Unlock()

	if r.status.Code != Running {
		// add all local service routes into the routing table
		if err := r.manageRegistryRoutes(r.opts.Registry, "insert"); err != nil {
			return nil, fmt.Errorf("failed adding routes: %s", err)
		}
		log.Logf("Routing table:\n%s", r.opts.Table)

		// list routing table routes to announce
		routes, err := r.opts.Table.List()
		if err != nil {
			return nil, fmt.Errorf("failed listing routes: %s", err)
		}
		// collect all the added routes before we attempt to add default gateway
		events := make([]*table.Event, len(routes))
		for i, route := range routes {
			event := &table.Event{
				Type:      table.Insert,
				Timestamp: time.Now(),
				Route:     route,
			}
			events[i] = event
		}

		// add default gateway into routing table
		if r.opts.Gateway != "" {
			// note, the only non-default value is the gateway
			route := table.Route{
				Service: "*",
				Address: "*",
				Gateway: r.opts.Gateway,
				Network: "*",
				Metric:  table.DefaultLocalMetric,
			}
			if err := r.opts.Table.Add(route); err != nil {
				return nil, fmt.Errorf("failed adding default gateway route: %s", err)
			}
		}

		// NOTE: we only need to recreate these if the router errored or was stopped
		// TODO: These probably dont need to be struct members
		if r.status.Code == Error || r.status.Code == Stopped {
			r.exit = make(chan struct{})
			r.eventChan = make(chan *table.Event)
			r.advertChan = make(chan *Advert)
		}

		// routing table watcher
		tableWatcher, err := r.opts.Table.Watch()
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

		// watch for errors and cleanup
		r.wg.Add(1)
		go r.watchErrors(errChan)

		// advertise your presence
		r.advertWg.Add(1)
		go r.advertEvents(Announce, events)

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
	events := make([]*table.Event, len(a.Events))
	copy(events, a.Events)
	// sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	for _, event := range events {
		// create a copy of the route
		route := event.Route
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
