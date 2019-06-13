package router

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/registry"
	"github.com/olekukonko/tablewriter"
)

type router struct {
	opts Options
	exit chan struct{}
	wg   *sync.WaitGroup
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

// Table returns routing table
func (r *router) Table() Table {
	return r.opts.Table
}

// Address returns router's bind address
func (r *router) Address() string {
	return r.opts.Address
}

// Gossip returns gossip bind address
func (r *router) Gossip() string {
	return r.opts.GossipAddress
}

// Network returns router's micro network
func (r *router) Network() string {
	return r.opts.NetworkAddress
}

// Start starts the router
func (r *router) Start() error {
	// add local service routes into the routing table
	if err := r.addServiceRoutes(r.opts.LocalRegistry, "local", 1); err != nil {
		return fmt.Errorf("failed adding routes for local services: %v", err)
	}

	// add network service routes into the routing table
	if err := r.addServiceRoutes(r.opts.NetworkRegistry, r.opts.NetworkAddress, 10); err != nil {
		return fmt.Errorf("failed adding routes for network services: %v", err)
	}

	// lookup local service routes and advertise them in network registry
	query := NewQuery(QueryNetwork("local"))
	localRoutes, err := r.opts.Table.Lookup(query)
	if err != nil && err != ErrRouteNotFound {
		return fmt.Errorf("failed to lookup local service routes: %v", err)
	}

	addr := strings.Split(r.opts.Address, ":")
	port, err := strconv.Atoi(addr[1])
	if err != nil {
		fmt.Errorf("could not parse router address from %s: %v", r.opts.Address, err)
	}

	for _, route := range localRoutes {
		node := &registry.Node{
			Id:      r.opts.ID,
			Address: addr[0],
			Port:    port,
		}

		service := &registry.Service{
			Name:  route.Options().DestAddr,
			Nodes: []*registry.Node{node},
		}
		if err := r.opts.NetworkRegistry.Register(service, registry.RegisterTTL(10*time.Second)); err != nil {
			return fmt.Errorf("failed to register service %s in network registry: %v", service.Name, err)
		}
	}

	lWatcher, err := r.opts.LocalRegistry.Watch()
	if err != nil {
		return fmt.Errorf("failed to create local registry watcher: %v", err)
	}

	rWatcher, err := r.opts.NetworkRegistry.Watch()
	if err != nil {
		return fmt.Errorf("failed to create network registry watcher: %v", err)
	}

	// we only watch local entries which we resend to network registry
	tWatcher, err := r.opts.Table.Watch(WatchNetwork("local"))
	if err != nil {
		return fmt.Errorf("failed to create routing table watcher: %v", err)
	}

	r.wg.Add(1)
	go r.manageServiceRoutes(lWatcher, "local", DefaultLocalMetric)

	r.wg.Add(1)
	go r.manageServiceRoutes(rWatcher, r.opts.NetworkAddress, DefaultNetworkMetric)

	r.wg.Add(1)
	go r.watchTable(tWatcher)

	return nil
}

// addServiceRouteslists all available services in given registry and adds them to the routing table.
// NOTE: this is a one-off operation done to bootstrap the rouing table of the new router when it starts.
// It returns error if the route could not be added to the routing table.
func (r *router) addServiceRoutes(reg registry.Registry, network string, metric int) error {
	services, err := reg.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %v", err)
	}

	for _, service := range services {
		route := NewRoute(
			DestAddr(service.Name),
			Gateway(r),
			Network(network),
			Metric(metric),
		)
		if err := r.opts.Table.Add(route); err != nil {
			return fmt.Errorf("failed to add route for service: %s", service.Name)
		}
	}

	return nil
}

// manageServiceRoutes watches services in given registry and updates the routing table accordingly.
// It returns error if the service registry watcher has stopped or if the routing table failed to be updated.
func (r *router) manageServiceRoutes(w registry.Watcher, network string, metric int) error {
	defer r.wg.Done()

	// wait in the background for the router to stop
	// when the router stops, stop the watcher and exit
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		<-r.exit
		w.Stop()
	}()

	var watchErr error

	// watch for changes to services
	for {
		res, err := w.Next()
		if err == registry.ErrWatcherStopped {
			break
		}

		if err != nil {
			watchErr = err
			break
		}

		route := NewRoute(
			DestAddr(res.Service.Name),
			Gateway(r),
			Network(network),
			Metric(metric),
		)

		switch res.Action {
		case "create":
			if len(res.Service.Nodes) > 0 {
				if err := r.opts.Table.Add(route); err != nil {
					return fmt.Errorf("failed to add route for service: %v", res.Service.Name)
				}
			}
		case "delete":
			if err := r.opts.Table.Remove(route); err != nil {
				return fmt.Errorf("failed to remove route for service: %v", res.Service.Name)
			}
		}
	}

	return watchErr
}

// watch routing table changes
func (r *router) watchTable(w Watcher) error {
	defer r.wg.Done()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		<-r.exit
		w.Stop()
	}()

	var watchErr error

	// watch for changes to services
	for {
		res, err := w.Next()
		if err == ErrWatcherStopped {
			break
		}

		if err != nil {
			watchErr = err
			break
		}

		addr := strings.Split(r.opts.Address, ":")
		port, err := strconv.Atoi(addr[1])
		if err != nil {
			continue
		}

		node := &registry.Node{
			Id:      r.opts.ID,
			Address: addr[0],
			Port:    port,
		}

		service := &registry.Service{
			Name:  res.Route.Options().DestAddr,
			Nodes: []*registry.Node{node},
		}

		switch res.Action {
		case "add":
			if err := r.opts.NetworkRegistry.Register(service, registry.RegisterTTL(10*time.Second)); err != nil {
				return fmt.Errorf("failed to register service %s in network registry: %v", service.Name, err)
			}
		case "remove":
			if err := r.opts.NetworkRegistry.Register(service); err != nil {
				return fmt.Errorf("failed to deregister service %s from network registry: %v", service.Name, err)
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
		r.opts.NetworkAddress,
		fmt.Sprintf("%d", r.opts.Table.Size()),
	}
	table.Append(data)

	// render table into sb
	table.Render()

	return sb.String()
}
