package router

import (
	"fmt"
	"strings"
	"sync"

	"github.com/micro/go-micro/registry"
	"github.com/olekukonko/tablewriter"
)

// router provides default router implementation
type router struct {
	opts Options
	exit chan struct{}
	wg   *sync.WaitGroup
}

// newRouter creates new router and returns it
func newRouter(opts ...Option) Router {
	// TODO: we need to add default GW entry here
	// Should default GW be part of router options?

	// get default options
	options := DefaultOptions()

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	return &router{
		opts: options,
		exit: make(chan struct{}),
		wg:   &sync.WaitGroup{},
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

// Advertise advertises the routes to the network. It is a blocking function.
// It returns error if any of the launched goroutines fail with error.
func (r *router) Advertise() error {
	// add local service routes into the routing table
	if err := r.addServiceRoutes(r.opts.Registry, "local", DefaultLocalMetric); err != nil {
		return fmt.Errorf("failed adding routes: %v", err)
	}

	localWatcher, err := r.opts.Registry.Watch()
	if err != nil {
		return fmt.Errorf("failed to create registry watcher: %v", err)
	}

	// error channel collecting goroutine errors
	errChan := make(chan error, 1)

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		// watch local registry and register routes in routine table
		errChan <- r.manageServiceRoutes(localWatcher, DefaultLocalMetric)
	}()

	return <-errChan
}

// addServiceRoutes adds all services in given registry to the routing table.
// NOTE: this is a one-off operation done when bootstrapping the routing table
// It returns error if either the services failed to be listed or
// if the routes could not be added to the routing table.
func (r *router) addServiceRoutes(reg registry.Registry, network string, metric int) error {
	services, err := reg.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %v", err)
	}

	// add each service node as a separate route;
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
			gw := node.Address
			if node.Port > 0 {
				gw = fmt.Sprintf("%s:%d", node.Address, node.Port)
			}
			route := Route{
				Destination: service.Name,
				Gateway:     gw,
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
		if err == registry.ErrWatcherStopped {
			break
		}

		if err != nil {
			watchErr = err
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

// Stop stops the router
func (r *router) Stop() error {
	// notify all goroutines to finish
	close(r.exit)

	// wait for all goroutines to finish
	r.wg.Wait()

	return nil
}

// String prints debugging information about router
func (r *router) String() string {
	sb := &strings.Builder{}

	table := tablewriter.NewWriter(sb)
	table.SetHeader([]string{"ID", "Address", "Network", "Table"})

	data := []string{
		r.opts.ID,
		r.opts.Address,
		r.opts.Network,
		fmt.Sprintf("%d", r.opts.Table.Size()),
	}
	table.Append(data)

	// render table into sb
	table.Render()

	return sb.String()
}
