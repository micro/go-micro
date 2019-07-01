package router

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/registry"
	"github.com/olekukonko/tablewriter"
)

// router provides default router implementation
type router struct {
	opts       Options
	status     Status
	advertChan chan *Update
	exit       chan struct{}
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
		advertChan: make(chan *Update),
		exit:       make(chan struct{}),
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
// NOTE: this is a one-off operation done when bootstrapping the routing table
// It returns error if either the services failed to be listed or
// if any of the the routes could not be added to the routing table.
func (r *router) addServiceRoutes(reg registry.Registry, network string, metric int) error {
	services, err := reg.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %v", err)
	}

	// add each service node as a separate route
	for _, service := range services {
		// get the service to retrieve all its info
		srvs, err := reg.GetService(service.Name)
		if err != nil {
			continue
		}

		// create a flat slide of nodes
		var nodes []*registry.Node
		for _, s := range srvs {
			nodes = append(nodes, s.Nodes...)
		}

		// range over the flat slice of nodes
		for _, node := range nodes {
			gateway := node.Address
			if node.Port > 0 {
				gateway = fmt.Sprintf("%s:%d", node.Address, node.Port)
			}
			route := Route{
				Destination: service.Name,
				Gateway:     gateway,
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

// manageServiceRoutes watches services in given registry and updates the routing table accordingly.
// It returns error if the service registry watcher has stopped or if the routing table failed to be updated.
func (r *router) manageServiceRoutes(w registry.Watcher, metric int) error {
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

		route := Route{
			Destination: res.Service.Name,
			Router:      r.opts.Address,
			Network:     r.opts.Network,
			Metric:      metric,
		}

		switch res.Action {
		case "create":
			// only return error if the route is not duplicate, but something else has failed
			if err := r.opts.Table.Add(route); err != nil && err != ErrDuplicateRoute {
				return fmt.Errorf("failed to add route for service %v: %s", res.Service.Name, err)
			}
		case "delete":
			// only return error if the route is not in the table, but something else has failed
			if err := r.opts.Table.Delete(route); err != nil && err != ErrRouteNotFound {
				return fmt.Errorf("failed to delete route for service %v: %s", res.Service.Name, err)
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

exit:
	for {
		event, err := w.Next()
		if err != nil {
			if err != ErrWatcherStopped {
				watchErr = err
			}
			break
		}

		u := &Update{
			ID:        r.ID(),
			Timestamp: time.Now(),
			Event:     event,
		}

		select {
		case <-r.exit:
			break exit
		case r.advertChan <- u:
		}
	}

	// close the advertisement channel
	close(r.advertChan)

	return watchErr
}

// watchError watches router errors
func (r *router) watchError(errChan <-chan error) {
	defer r.wg.Done()

	var code StatusCode
	var err error

	select {
	case <-r.exit:
		code = Stopped
	case err = <-errChan:
		code = Error
	}

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
	}
}

// Advertise advertises the routes to the network.
// It returns error if any of the launched goroutines fail with error.
func (r *router) Advertise() (<-chan *Update, error) {
	r.Lock()
	defer r.Unlock()

	if r.status.Code != Running {
		// add local service routes into the routing table
		if err := r.addServiceRoutes(r.opts.Registry, "local", DefaultLocalMetric); err != nil {
			return nil, fmt.Errorf("failed adding routes: %v", err)
		}
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
				return nil, fmt.Errorf("error to add default gateway route: %s", err)
			}
		}

		// NOTE: we only need to recreate the exit/advertChan if the router errored or was stopped
		if r.status.Code == Error || r.status.Code == Stopped {
			r.exit = make(chan struct{})
			r.advertChan = make(chan *Update)
		}

		// routing table watcher which watches all routes i.e. to every destination
		tableWatcher, err := r.opts.Table.Watch(WatchDestination("*"))
		if err != nil {
			return nil, fmt.Errorf("failed to create routing table watcher: %v", err)
		}
		// registry watcher
		regWatcher, err := r.opts.Registry.Watch()
		if err != nil {
			return nil, fmt.Errorf("failed to create registry watcher: %v", err)
		}

		// error channel collecting goroutine errors
		errChan := make(chan error, 2)

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			// watch local registry and register routes in routine table
			errChan <- r.manageServiceRoutes(regWatcher, DefaultLocalMetric)
		}()

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			// watch local registry and register routes in routing table
			errChan <- r.watchTable(tableWatcher)
		}()

		r.wg.Add(1)
		go r.watchError(errChan)

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
func (r *router) Update(a *Update) error {
	// we extract the route from advertisement and update the routing table
	route := Route{
		Destination: a.Event.Route.Destination,
		Gateway:     a.Event.Route.Gateway,
		Router:      a.Event.Route.Router,
		Network:     a.Event.Route.Network,
		Metric:      a.Event.Route.Metric,
		Policy:      AddIfNotExists,
	}

	return r.opts.Table.Update(route)
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
	r.RLock()
	// only close the channel if the router is running
	if r.status.Code == Running {
		// notify all goroutines to finish
		close(r.exit)
		// drain the advertise channel
		for range r.advertChan {
		}
	}
	r.RUnlock()

	// wait for all goroutines to finish
	r.wg.Wait()

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
