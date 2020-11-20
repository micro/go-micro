package registry

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/asim/nitro/app/logger"
	"github.com/asim/nitro/app/registry"
	"github.com/asim/nitro/app/router"
)

var (
	// RefreshInterval is the time at which we completely refresh the table
	RefreshInterval = time.Second * 120
	// PruneInterval is how often we prune the routing table
	PruneInterval = time.Second * 10
)

// rtr implements router interface
type rtr struct {
	sync.RWMutex

	running  bool
	table    *table
	options  router.Options
	exit     chan bool
	initChan chan bool
}

// NewRouter creates new router and returns it
func NewRouter(opts ...router.Option) router.Router {
	// get default options
	options := router.DefaultOptions()

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// construct the router
	r := &rtr{
		options:  options,
		initChan: make(chan bool),
	}

	// create the new table, passing the fetchRoute method in as a fallback if
	// the table doesn't contain the result for a query.
	r.table = newTable()

	// start the router
	r.start()
	return r
}

// Init initializes router with given options
func (r *rtr) Init(opts ...router.Option) error {
	r.Lock()
	for _, o := range opts {
		o(&r.options)
	}
	r.Unlock()

	// push a message to the init chan so the watchers
	// can reset in the case the registry was changed
	go func() {
		r.initChan <- true
	}()

	return nil
}

// Options returns router options
func (r *rtr) Options() router.Options {
	r.RLock()
	defer r.RUnlock()

	options := r.options

	return options
}

// Table returns routing table
func (r *rtr) Table() router.Table {
	r.Lock()
	defer r.Unlock()
	return r.table
}

func getDomain(srv *registry.Service) string {
	// check the service metadata for domain
	// TODO: domain as Domain field in registry?
	if srv.Metadata != nil && len(srv.Metadata["domain"]) > 0 {
		return srv.Metadata["domain"]
	} else if len(srv.Nodes) > 0 && srv.Nodes[0].Metadata != nil {
		return srv.Nodes[0].Metadata["domain"]
	}

	// otherwise return wildcard
	// TODO: return GlobalDomain or PublicDomain
	return registry.DefaultDomain
}

// manageRoute applies action on a given route
func (r *rtr) manageRoute(route router.Route, action string) error {
	switch action {
	case "create":
		if err := r.table.Create(route); err != nil && err != router.ErrDuplicateRoute {
			return fmt.Errorf("failed adding route for service %s: %s", route.Service, err)
		}
	case "delete":
		if err := r.table.Delete(route); err != nil && err != router.ErrRouteNotFound {
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

// createRoutes turns a service into a list routes basically converting nodes to routes
func (r *rtr) createRoutes(service *registry.Service, network string) []router.Route {
	var routes []router.Route

	for _, node := range service.Nodes {
		routes = append(routes, router.Route{
			Service:  service.Name,
			Address:  node.Address,
			Gateway:  "",
			Network:  network,
			Router:   r.options.Id,
			Link:     router.DefaultLink,
			Metric:   router.DefaultMetric,
			Metadata: node.Metadata,
		})
	}

	return routes
}

// manageServiceRoutes applies action to all routes of the service.
// It returns error of the action fails with error.
func (r *rtr) manageRoutes(service *registry.Service, action, network string) error {
	// action is the routing table action
	action = strings.ToLower(action)

	// create a set of routes from the service
	routes := r.createRoutes(service, network)

	// if its a delete action and there's no nodes
	// it means we need to wipe out all the routes
	// for that service
	if action == "delete" && len(routes) == 0 {
		// delete the service entirely
		r.table.deleteService(service.Name, network)
		return nil
	}

	// create the routes in the table
	for _, route := range routes {
		logger.Tracef("Creating route %v domain: %v", route, network)
		if err := r.manageRoute(route, action); err != nil {
			return err
		}
	}

	return nil
}

// manageRegistryRoutes applies action to all routes of each service found in the registry.
// It returns error if either the services failed to be listed or the routing table action fails.
func (r *rtr) loadRoutes(reg registry.Registry) error {
	services, err := reg.ListServices(registry.ListDomain(registry.WildcardDomain))
	if err != nil {
		return fmt.Errorf("failed listing services: %v", err)
	}

	// add each service node as a separate route
	for _, service := range services {
		// get the services domain from metadata. Fallback to wildcard.
		domain := getDomain(service)

		// create the routes
		routes := r.createRoutes(service, domain)

		// if the routes exist save them
		if len(routes) > 0 {
			logger.Tracef("Creating routes for service %v domain: %v", service, domain)
			for _, rt := range routes {
				err := r.table.Create(rt)

				// update the route to prevent it from expiring
				if err == router.ErrDuplicateRoute {
					err = r.table.Update(rt)
				}

				if err != nil {
					logger.Errorf("Error creating route for service %v in domain %v: %v", service, domain, err)
				}
			}
			continue
		}

		// otherwise get all the service info

		// get the service to retrieve all its info
		srvs, err := reg.GetService(service.Name, registry.GetDomain(domain))
		if err != nil {
			logger.Tracef("Failed to get service %s domain: %s", service.Name, domain)
			continue
		}

		// manage the routes for all returned services
		for _, srv := range srvs {
			routes := r.createRoutes(srv, domain)

			if len(routes) > 0 {
				logger.Tracef("Creating routes for service %v domain: %v", srv, domain)
				for _, rt := range routes {
					err := r.table.Create(rt)

					// update the route to prevent it from expiring
					if err == router.ErrDuplicateRoute {
						err = r.table.Update(rt)
					}

					if err != nil {
						logger.Errorf("Error creating route for service %v in domain %v: %v", service, domain, err)
					}
				}
			}
		}
	}

	return nil
}

// Close the router
func (r *rtr) Close() error {
	r.Lock()
	defer r.Unlock()

	select {
	case <-r.exit:
		return nil
	default:
		if !r.running {
			return nil
		}
		close(r.exit)

	}

	r.running = false
	return nil
}

// lookup retrieves all the routes for a given service and creates them in the routing table
func (r *rtr) Lookup(service string, opts ...router.LookupOption) ([]router.Route, error) {
	q := router.NewLookup(opts...)

	// if we find the routes filter and return them
	routes, err := r.table.Read(router.ReadService(service))
	if err == nil {
		routes = router.Filter(routes, q)
		if len(routes) == 0 {
			return nil, router.ErrRouteNotFound
		}
		return routes, nil
	}

	// lookup the route
	logger.Tracef("Fetching route for %s domain: %v", service, registry.WildcardDomain)

	services, err := r.options.Registry.GetService(service, registry.GetDomain(registry.WildcardDomain))
	if err == registry.ErrNotFound {
		logger.Tracef("Failed to find route for %s", service)
		return nil, router.ErrRouteNotFound
	} else if err != nil {
		logger.Tracef("Failed to find route for %s: %v", service, err)
		return nil, fmt.Errorf("failed getting services: %v", err)
	}

	for _, srv := range services {
		domain := getDomain(srv)
		// TODO: should we continue to send the event indicating we created a route?
		// lookup is only called in the query path so probably not
		routes = append(routes, r.createRoutes(srv, domain)...)
	}

	// if we're supposed to cache then save the routes
	if r.options.Cache {
		for _, route := range routes {
			r.table.Create(route)
		}
	}

	routes = router.Filter(routes, q)
	if len(routes) == 0 {
		return nil, router.ErrRouteNotFound
	}
	return routes, nil
}

// watchRegistry watches registry and updates routing table based on the received events.
// It returns error if either the registry watcher fails with error or if the routing table update fails.
func (r *rtr) watchRegistry(w registry.Watcher) error {
	exit := make(chan bool)

	defer func() {
		close(exit)
	}()

	go func() {
		defer w.Stop()

		select {
		case <-exit:
			return
		case <-r.initChan:
			return
		case <-r.exit:
			return
		}
	}()

	for {
		// get the next service
		res, err := w.Next()
		if err != nil {
			if err != registry.ErrWatcherStopped {
				return err
			}
			break
		}

		// don't process nil entries
		if res.Service == nil {
			logger.Trace("Received a nil service")
			continue
		}

		logger.Tracef("Router dealing with next route %s %+v\n", res.Action, res.Service)

		// get the services domain from metadata. Fallback to wildcard.
		domain := getDomain(res.Service)

		// create/update or delete the route
		if err := r.manageRoutes(res.Service, res.Action, domain); err != nil {
			return err
		}
	}

	return nil
}

// start the router. Should be called under lock.
func (r *rtr) start() error {
	if r.running {
		return nil
	}

	// add default gateway into routing table
	if r.options.Gateway != "" {
		// note, the only non-default value is the gateway
		route := router.Route{
			Service: "*",
			Address: "*",
			Gateway: r.options.Gateway,
			Network: "*",
			Router:  r.options.Id,
			Link:    router.DefaultLink,
			Metric:  router.DefaultMetric,
		}
		if err := r.table.Create(route); err != nil {
			return fmt.Errorf("failed adding default gateway route: %s", err)
		}
	}

	// create error and exit channels
	r.exit = make(chan bool)
	r.running = true

	// only cache if told to do so
	if !r.options.Cache {
		return nil
	}

	// create a refresh notify channel
	refresh := make(chan bool, 1)

	// fires the refresh for loading routes
	refreshRoutes := func() {
		select {
		case refresh <- true:
		default:
		}
	}

	// refresh all the routes in the event of a failure watching the registry
	go func() {
		var lastRefresh time.Time

		// load a refresh
		refreshRoutes()

		for {
			select {
			case <-r.exit:
				return
			case <-refresh:
				// don't refresh if we've done so in the past minute
				if !lastRefresh.IsZero() && time.Since(lastRefresh) < time.Minute {
					continue
				}

				// load new routes
				if err := r.loadRoutes(r.options.Registry); err != nil {
					logger.Debugf("failed refreshing registry routes: %s", err)
					// in this don't prune
					continue
				}

				// first time so nothing to prune
				if !lastRefresh.IsZero() {
					// prune any routes since last refresh since we've
					// updated basically everything we care about
					r.table.pruneRoutes(time.Since(lastRefresh))
				}

				// update the refresh time
				lastRefresh = time.Now()
			case <-time.After(RefreshInterval):
				refreshRoutes()
			}
		}
	}()

	go func() {
		for {
			select {
			case <-r.exit:
				return
			default:
				logger.Tracef("Router starting registry watch")
				w, err := r.options.Registry.Watch(registry.WatchDomain(registry.WildcardDomain))
				if err != nil {
					if logger.V(logger.DebugLevel, logger.DefaultLogger) {
						logger.Debugf("failed creating registry watcher: %v", err)
					}
					time.Sleep(time.Second)
					// in the event of an error reload routes
					refreshRoutes()
					continue
				}

				// watchRegistry calls stop when it's done
				if err := r.watchRegistry(w); err != nil {
					if logger.V(logger.DebugLevel, logger.DefaultLogger) {
						logger.Debugf("Error watching the registry: %v", err)
					}
					time.Sleep(time.Second)
					// in the event of an error reload routes
					refreshRoutes()
				}
			}
		}
	}()

	return nil
}

// Watch routes
func (r *rtr) Watch(opts ...router.WatchOption) (router.Watcher, error) {
	return r.table.Watch(opts...)
}

// String prints debugging information about router
func (r *rtr) String() string {
	return "registry"
}
