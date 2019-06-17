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
	if err := r.addServiceRoutes(r.opts.LocalRegistry, "local", DefaultLocalMetric); err != nil {
		return fmt.Errorf("failed adding routes for local services: %v", err)
	}

	// add network service routes into the routing table
	if err := r.addServiceRoutes(r.opts.NetworkRegistry, r.opts.NetworkAddress, DefaultNetworkMetric); err != nil {
		return fmt.Errorf("failed adding routes for network services: %v", err)
	}

	// routing table has been bootstrapped;
	// NOTE: we only need to advertise local services upstream
	// lookup local service routes and advertise them upstream
	query := NewQuery(QueryNetwork("local"))
	localRoutes, err := r.opts.Table.Lookup(query)
	if err != nil && err != ErrRouteNotFound {
		return fmt.Errorf("failed to lookup local service routes: %v", err)
	}

	node, err := r.parseToNode()
	if err != nil {
		return fmt.Errorf("failed to parse router into service node: %v", err)
	}

	for _, route := range localRoutes {
		service := &registry.Service{
			Name:  route.Options().DestAddr,
			Nodes: []*registry.Node{node},
		}
		if err := r.opts.NetworkRegistry.Register(service, registry.RegisterTTL(120*time.Second)); err != nil {
			return fmt.Errorf("failed to register service %s in network registry: %v", service.Name, err)
		}
	}

	localWatcher, err := r.opts.LocalRegistry.Watch()
	if err != nil {
		return fmt.Errorf("failed to create local registry watcher: %v", err)
	}

	networkWatcher, err := r.opts.NetworkRegistry.Watch()
	if err != nil {
		return fmt.Errorf("failed to create network registry watcher: %v", err)
	}

	// we only watch local netwrork entries which we then propagate upstream to network
	tableWatcher, err := r.opts.Table.Watch(WatchNetwork("local"))
	if err != nil {
		return fmt.Errorf("failed to create routing table watcher: %v", err)
	}

	r.wg.Add(1)
	go r.manageServiceRoutes(localWatcher, "local", DefaultLocalMetric)

	r.wg.Add(1)
	go r.manageServiceRoutes(networkWatcher, r.opts.NetworkAddress, DefaultNetworkMetric)

	r.wg.Add(1)
	go r.watchTable(tableWatcher)

	return nil
}

// addServiceRouteslists all available services in given registry and adds them to the routing table.
// NOTE: this is a one-off operation done when bootstrapping the routing table of the new router.
// It returns error if either the services could not be listed or if the routes could not be added to the routing table.
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
		if err := r.opts.Table.Add(route); err != nil && err != ErrDuplicateRoute {
			return fmt.Errorf("error adding route for service: %s", service.Name)
		}
	}

	return nil
}

// parseToNode parses router into registry.Node and returns the result.
// It returns error if the router network address could not be parsed into service host and port.
// NOTE: We use ":" as the default delimiter when we split the network address.
func (r *router) parseToNode() (*registry.Node, error) {
	// split on ":" as a standard host/port delimiter
	addr := strings.Split(r.opts.NetworkAddress, ":")
	// try to parse network port into integer
	port, err := strconv.Atoi(addr[1])
	if err != nil {
		return nil, fmt.Errorf("could not parse router network address from %s: %v", r.opts.NetworkAddress, err)
	}

	node := &registry.Node{
		Id:      r.opts.ID,
		Address: addr[0],
		Port:    port,
	}

	return node, nil
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
				/// only return error if the route is not duplicate, but something else has failed
				if err := r.opts.Table.Add(route); err != nil && err != ErrDuplicateRoute {
					return fmt.Errorf("failed to add route for service: %v", res.Service.Name)
				}
			}
		case "delete":
			if len(res.Service.Nodes) <= 1 {
				// only return error if the route is present in the table, but something else has failed
				if err := r.opts.Table.Delete(route); err != nil && err != ErrRouteNotFound {
					return fmt.Errorf("failed to delete route for service: %v", res.Service.Name)
				}
			}
		}
	}

	return watchErr
}

// watchTable watches routing table entries and either adds or deletes locally registered service to/from network registry
// It returns error if the locally registered services either fails to be added/deleted to/from network registry.
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

		node, err := r.parseToNode()
		if err != nil {
			return fmt.Errorf("failed to parse router into node: %v", err)
		}

		service := &registry.Service{
			Name:  res.Route.Options().DestAddr,
			Nodes: []*registry.Node{node},
		}

		switch res.Action {
		case "add":
			// only register remotely if the service is "local"
			if res.Route.Options().Network == "local" {
				if err := r.opts.NetworkRegistry.Register(service, registry.RegisterTTL(120*time.Second)); err != nil {
					return fmt.Errorf("failed to register service %s in network registry: %v", service.Name, err)
				}
			}
		case "delete":
			// only deregister remotely if the service is "local"
			if res.Route.Options().Network == "local" {
				if err := r.opts.NetworkRegistry.Deregister(service); err != nil {
					return fmt.Errorf("failed to deregister service %s from network registry: %v", service.Name, err)
				}
			}
		}
	}

	return watchErr
}

// Stop stops the router
func (r *router) Stop() error {
	// NOTE: we need a more efficient way of doing this e.g. network routes
	// should ideally be autodeleted when the router stops gossiping
	// deregister all services advertised by this router from remote registry
	query := NewQuery(QueryGateway(r), QueryNetwork(r.opts.NetworkAddress))
	routes, err := r.opts.Table.Lookup(query)
	if err != nil && err != ErrRouteNotFound {
		return fmt.Errorf("failed to lookup routes for router %s: %v", r.opts.ID, err)
	}

	node, err := r.parseToNode()
	if err != nil {
		return fmt.Errorf("failed to parse router into service node: %v", err)
	}

	for _, route := range routes {
		service := &registry.Service{
			Name:  route.Options().DestAddr,
			Nodes: []*registry.Node{node},
		}
		if err := r.opts.NetworkRegistry.Deregister(service); err != nil {
			return fmt.Errorf("failed to deregister service %s from network registry: %v", service.Name, err)
		}
	}

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
