package selector

import (
	"sync"
	"time"

	"github.com/micro/go-log"
	"github.com/micro/go-micro/registry"
)

type registrySelector struct {
	so  Options
	ttl time.Duration

	// registry cache
	sync.RWMutex
	cache map[string][]*registry.Service
	ttls  map[string]time.Time

	watched map[string]bool

	// used to close or reload watcher
	reload chan bool
	exit   chan bool
}

var (
	DefaultTTL = time.Minute
)

// isValid checks if the service is valid
func (c *registrySelector) isValid(services []*registry.Service, ttl time.Time) bool {
	// no services exist
	if len(services) == 0 {
		return false
	}

	// ttl is invalid
	if ttl.IsZero() {
		return false
	}

	// time since ttl is longer than timeout
	if time.Since(ttl) > c.ttl {
		return false
	}

	// ok
	return true
}

func (c *registrySelector) quit() bool {
	select {
	case <-c.exit:
		return true
	default:
		return false
	}
}

// cp copies a service. Because we're caching handing back pointers would
// create a race condition, so we do this instead
// its fast enough
func (c *registrySelector) cp(current []*registry.Service) []*registry.Service {
	var services []*registry.Service

	for _, service := range current {
		// copy service
		s := new(registry.Service)
		*s = *service

		// copy nodes
		var nodes []*registry.Node
		for _, node := range service.Nodes {
			n := new(registry.Node)
			*n = *node
			nodes = append(nodes, n)
		}
		s.Nodes = nodes

		// copy endpoints
		var eps []*registry.Endpoint
		for _, ep := range service.Endpoints {
			e := new(registry.Endpoint)
			*e = *ep
			eps = append(eps, e)
		}
		s.Endpoints = eps

		// append service
		services = append(services, s)
	}

	return services
}

func (c *registrySelector) del(service string) {
	delete(c.cache, service)
	delete(c.ttls, service)
}

func (c *registrySelector) get(service string) ([]*registry.Service, error) {
	// read lock
	c.RLock()

	// check the cache first
	services := c.cache[service]
	// get cache ttl
	ttl := c.ttls[service]

	// got services && within ttl so return cache
	if c.isValid(services, ttl) {
		// make a copy
		cp := c.cp(services)
		// unlock the read
		c.RUnlock()
		// return servics
		return cp, nil
	}

	// get does the actual request for a service and cache it
	get := func(service string) ([]*registry.Service, error) {
		// ask the registry
		services, err := c.so.Registry.GetService(service)
		if err != nil {
			return nil, err
		}

		// cache results
		c.Lock()
		c.set(service, c.cp(services))
		c.Unlock()

		return services, nil
	}

	// watch service if not watched
	if _, ok := c.watched[service]; !ok {
		go c.run(service)
	}

	// unlock the read lock
	c.RUnlock()

	// get and return services
	return get(service)
}

func (c *registrySelector) set(service string, services []*registry.Service) {
	c.cache[service] = services
	c.ttls[service] = time.Now().Add(c.ttl)
}

func (c *registrySelector) update(res *registry.Result) {
	if res == nil || res.Service == nil {
		return
	}

	c.Lock()
	defer c.Unlock()

	services, ok := c.cache[res.Service.Name]
	if !ok {
		// we're not going to cache anything
		// unless there was already a lookup
		return
	}

	if len(res.Service.Nodes) == 0 {
		switch res.Action {
		case "delete":
			c.del(res.Service.Name)
		}
		return
	}

	// existing service found
	var service *registry.Service
	var index int
	for i, s := range services {
		if s.Version == res.Service.Version {
			service = s
			index = i
		}
	}

	switch res.Action {
	case "create", "update":
		if service == nil {
			c.set(res.Service.Name, append(services, res.Service))
			return
		}

		// append old nodes to new service
		for _, cur := range service.Nodes {
			var seen bool
			for _, node := range res.Service.Nodes {
				if cur.Id == node.Id {
					seen = true
					break
				}
			}
			if !seen {
				res.Service.Nodes = append(res.Service.Nodes, cur)
			}
		}

		services[index] = res.Service
		c.set(res.Service.Name, services)
	case "delete":
		if service == nil {
			return
		}

		var nodes []*registry.Node

		// filter cur nodes to remove the dead one
		for _, cur := range service.Nodes {
			var seen bool
			for _, del := range res.Service.Nodes {
				if del.Id == cur.Id {
					seen = true
					break
				}
			}
			if !seen {
				nodes = append(nodes, cur)
			}
		}

		// still got nodes, save and return
		if len(nodes) > 0 {
			service.Nodes = nodes
			services[index] = service
			c.set(service.Name, services)
			return
		}

		// zero nodes left

		// only have one thing to delete
		// nuke the thing
		if len(services) == 1 {
			c.del(service.Name)
			return
		}

		// still have more than 1 service
		// check the version and keep what we know
		var srvs []*registry.Service
		for _, s := range services {
			if s.Version != service.Version {
				srvs = append(srvs, s)
			}
		}

		// save
		c.set(service.Name, srvs)
	}
}

// run starts the cache watcher loop
// it creates a new watcher if there's a problem
// reloads the watcher if Init is called
// and returns when Close is called
func (c *registrySelector) run(name string) {
	// set watcher
	c.Lock()
	c.watched[name] = true
	c.Unlock()

	// delete watcher on exit
	defer func() {
		c.Lock()
		delete(c.watched, name)
		c.Unlock()
	}()

	// error counter
	var cerr int

	for {
		// exit early if already dead
		if c.quit() {
			return
		}

		// create new watcher
		w, err := c.so.Registry.Watch(
			registry.WatchService(name),
		)

		if err != nil {
			if c.quit() {
				return
			}
			cerr++
			if cerr > 3 {
				log.Log(err)
				cerr = 0
			}
			time.Sleep(time.Second)
			continue
		}

		// watch for events
		if err := c.watch(w); err != nil {
			if c.quit() {
				return
			}
			cerr++
			if cerr > 3 {
				cerr = 0
				log.Log(err)
			}
			continue
		}

		// reset err counter
		cerr = 0
	}
}

// watch loops the next event and calls update
// it returns if there's an error
func (c *registrySelector) watch(w registry.Watcher) error {
	defer w.Stop()

	// reload chan
	reload := make(chan bool, 1)

	// manage this loop
	go func() {
		// wait for exit or reload signal
		select {
		case <-c.exit:
		case <-c.reload:
			reload <- true
		}

		// stop the watcher
		w.Stop()
	}()

	for {
		res, err := w.Next()
		if err != nil {
			select {
			case <-reload:
				return nil
			default:
				return err
			}
		}
		c.update(res)
	}
}

func (c *registrySelector) Init(opts ...Option) error {
	for _, o := range opts {
		o(&c.so)
	}

	// reload the watcher
	go func() {
		select {
		case <-c.exit:
			return
		default:
			c.reload <- true
		}
	}()

	return nil
}

func (c *registrySelector) Options() Options {
	return c.so
}

func (c *registrySelector) Select(service string, opts ...SelectOption) (Next, error) {
	sopts := SelectOptions{
		Strategy: c.so.Strategy,
	}

	for _, opt := range opts {
		opt(&sopts)
	}

	// get the service
	// try the cache first
	// if that fails go directly to the registry
	services, err := c.get(service)
	if err != nil {
		return nil, err
	}

	// apply the filters
	for _, filter := range sopts.Filters {
		services = filter(services)
	}

	// if there's nothing left, return
	if len(services) == 0 {
		return nil, ErrNoneAvailable
	}

	return sopts.Strategy(services), nil
}

func (c *registrySelector) Mark(service string, node *registry.Node, err error) {
}

func (c *registrySelector) Reset(service string) {
}

// Close stops the watcher and destroys the cache
func (c *registrySelector) Close() error {
	c.Lock()
	c.cache = make(map[string][]*registry.Service)
	c.watched = make(map[string]bool)
	c.Unlock()

	select {
	case <-c.exit:
		return nil
	default:
		close(c.exit)
	}
	return nil
}

func (c *registrySelector) String() string {
	return "registry"
}

func NewSelector(opts ...Option) Selector {
	sopts := Options{
		Strategy: Random,
	}

	for _, opt := range opts {
		opt(&sopts)
	}

	if sopts.Registry == nil {
		sopts.Registry = registry.DefaultRegistry
	}

	ttl := DefaultTTL

	if sopts.Context != nil {
		if t, ok := sopts.Context.Value("selector_ttl").(time.Duration); ok {
			ttl = t
		}
	}

	return &registrySelector{
		so:      sopts,
		ttl:     ttl,
		watched: make(map[string]bool),
		cache:   make(map[string][]*registry.Service),
		ttls:    make(map[string]time.Time),
		reload:  make(chan bool, 1),
		exit:    make(chan bool),
	}
}
