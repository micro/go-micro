// Package cache provides a registry cache
package cache

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/registry"
	util "github.com/micro/go-micro/v3/util/registry"
)

// Cache is the registry cache interface
type Cache interface {
	// embed the registry interface
	registry.Registry
	// stop the cache watcher
	Stop()
}

type Options struct {
	// TTL is the cache TTL
	TTL time.Duration
}

type Option func(o *Options)

type cache struct {
	registry.Registry
	opts Options

	// registry cache. services,ttls,watched,running are grouped by doman
	sync.RWMutex
	services map[string]services
	ttls     map[string]ttls
	watched  map[string]watched
	running  map[string]bool

	// used to stop the caches
	exit chan bool

	// indicate whether its running status of the registry used to hold onto the cache in failure state
	status error
}

type services map[string][]*registry.Service
type ttls map[string]time.Time
type watched map[string]bool

var defaultTTL = time.Minute

func backoff(attempts int) time.Duration {
	if attempts == 0 {
		return time.Duration(0)
	}
	return time.Duration(math.Pow(10, float64(attempts))) * time.Millisecond
}

func (c *cache) getStatus() error {
	c.RLock()
	defer c.RUnlock()
	return c.status
}

func (c *cache) setStatus(err error) {
	c.Lock()
	c.status = err
	c.Unlock()
}

// isValid checks if the service is valid
func (c *cache) isValid(services []*registry.Service, ttl time.Time) bool {
	// no services exist
	if len(services) == 0 {
		return false
	}

	// ttl is invalid
	if ttl.IsZero() {
		return false
	}

	// time since ttl is longer than timeout
	if time.Since(ttl) > 0 {
		return false
	}

	// ok
	return true
}

func (c *cache) quit() bool {
	select {
	case <-c.exit:
		return true
	default:
		return false
	}
}

func (c *cache) del(domain, service string) {
	// don't blow away cache in error state
	if err := c.getStatus(); err != nil {
		return
	}

	c.Lock()
	defer c.Unlock()

	if _, ok := c.services[domain]; ok {
		delete(c.services[domain], service)
	}

	if _, ok := c.ttls[domain]; ok {
		delete(c.ttls[domain], service)
	}
}

func (c *cache) get(domain, service string) ([]*registry.Service, error) {
	var services []*registry.Service
	var ttl time.Time

	// lookup the values in the cache before calling the underlying registrry
	c.RLock()
	if srvs, ok := c.services[domain]; ok {
		services = srvs[service]
	}
	if tt, ok := c.ttls[domain]; ok {
		ttl = tt[service]
	}
	c.RUnlock()

	// got services && within ttl so return a copy of the services
	if c.isValid(services, ttl) {
		return util.Copy(services), nil
	}

	// get does the actual request for a service and cache it
	get := func(domain string, service string, cached []*registry.Service) ([]*registry.Service, error) {
		// ask the registry
		services, err := c.Registry.GetService(service, registry.GetDomain(domain))
		if err != nil {
			// set the error status
			c.setStatus(err)

			// check the cache
			if len(cached) > 0 {
				return cached, nil
			}

			// otherwise return error
			return nil, err
		}

		// reset the status
		if err := c.getStatus(); err != nil {
			c.setStatus(nil)
		}

		// cache results
		c.set(domain, service, util.Copy(services))

		return services, nil
	}

	// watch service if not watched
	c.RLock()
	var ok bool
	if _, d := c.watched[domain]; d {
		if _, s := c.watched[domain][service]; s {
			ok = true
		}
	}
	c.RUnlock()

	// check if its being watched
	if !ok {
		c.Lock()

		// add domain if not registered
		if _, ok := c.watched[domain]; !ok {
			c.watched[domain] = make(map[string]bool)
		}

		// set to watched
		c.watched[domain][service] = true

		running := c.running[domain]
		c.Unlock()

		// only kick it off if not running
		if !running {
			go c.run(domain)
		}
	}

	// get and return services
	return get(domain, service, services)
}

func (c *cache) set(domain string, service string, srvs []*registry.Service) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.services[domain]; !ok {
		c.services[domain] = make(services)
	}
	if _, ok := c.ttls[domain]; !ok {
		c.ttls[domain] = make(ttls)
	}

	c.services[domain][service] = srvs
	c.ttls[domain][service] = time.Now().Add(c.opts.TTL)
}

func (c *cache) update(domain string, res *registry.Result) {
	if res == nil || res.Service == nil {
		return
	}

	// only save watched services since the service using the cache may only depend on a handful
	// of other services
	c.RLock()
	if _, ok := c.watched[res.Service.Name]; !ok {
		c.RUnlock()
		return
	}

	// we're not going to cache anything unless there was already a lookup
	services, ok := c.services[domain][res.Service.Name]
	if !ok {
		c.RUnlock()
		return
	}

	c.RUnlock()

	if len(res.Service.Nodes) == 0 {
		switch res.Action {
		case "delete":
			c.del(domain, res.Service.Name)
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
			c.set(domain, res.Service.Name, append(services, res.Service))
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
		c.set(domain, res.Service.Name, services)
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
			c.set(domain, service.Name, services)
			return
		}

		// zero nodes left

		// only have one thing to delete
		// nuke the thing
		if len(services) == 1 {
			c.del(domain, service.Name)
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
		c.set(domain, service.Name, srvs)
	}
}

// run starts the cache watcher loop
// it creates a new watcher if there's a problem
func (c *cache) run(domain string) {
	c.Lock()
	c.running[domain] = true
	c.Unlock()

	// reset watcher on exit
	defer func() {
		c.Lock()
		c.watched[domain] = make(map[string]bool)
		c.running[domain] = false
		c.Unlock()
	}()

	var a, b int

	for {
		// exit early if already dead
		if c.quit() {
			return
		}

		// jitter before starting
		j := rand.Int63n(100)
		time.Sleep(time.Duration(j) * time.Millisecond)

		// create new watcher
		w, err := c.Registry.Watch(registry.WatchDomain(domain))
		if err != nil {
			if c.quit() {
				return
			}

			d := backoff(a)
			c.setStatus(err)

			if a > 3 {
				if logger.V(logger.DebugLevel, logger.DefaultLogger) {
					logger.Debug("rcache: ", err, " backing off ", d)
				}
				a = 0
			}

			time.Sleep(d)
			a++

			continue
		}

		// reset a
		a = 0

		// watch for events
		if err := c.watch(domain, w); err != nil {
			if c.quit() {
				return
			}

			d := backoff(b)
			c.setStatus(err)

			if b > 3 {
				if logger.V(logger.DebugLevel, logger.DefaultLogger) {
					logger.Debug("rcache: ", err, " backing off ", d)
				}
				b = 0
			}

			time.Sleep(d)
			b++

			continue
		}

		// reset b
		b = 0
	}
}

// watch loops the next event and calls update
// it returns if there's an error
func (c *cache) watch(domain string, w registry.Watcher) error {
	// used to stop the watch
	stop := make(chan bool)

	// manage this loop
	go func() {
		defer w.Stop()

		select {
		// wait for exit
		case <-c.exit:
			return
		// we've been stopped
		case <-stop:
			return
		}
	}()

	for {
		res, err := w.Next()
		if err != nil {
			close(stop)
			return err
		}

		// reset the error status since we succeeded
		if err := c.getStatus(); err != nil {
			// reset status
			c.setStatus(nil)
		}

		// for wildcard queries, the domain will be * and not the services domain, so we'll check to
		// see if it was provided in the metadata.
		dom := domain
		if res.Service.Metadata != nil && len(res.Service.Metadata["domain"]) > 0 {
			dom = res.Service.Metadata["domain"]
		}

		c.update(dom, res)
	}
}

func (c *cache) GetService(service string, opts ...registry.GetOption) ([]*registry.Service, error) {
	// parse the options, fallback to the default domain
	var options registry.GetOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.Domain) == 0 {
		options.Domain = registry.DefaultDomain
	}

	// get the service
	services, err := c.get(options.Domain, service)
	if err != nil {
		return nil, err
	}

	// if there's nothing return err
	if len(services) == 0 {
		return nil, registry.ErrNotFound
	}

	// return services
	return services, nil
}

func (c *cache) Stop() {
	c.Lock()
	defer c.Unlock()

	select {
	case <-c.exit:
		return
	default:
		close(c.exit)
	}
}

func (c *cache) String() string {
	return "cache"
}

// New returns a new cache
func New(r registry.Registry, opts ...Option) Cache {
	rand.Seed(time.Now().UnixNano())
	options := Options{
		TTL: defaultTTL,
	}

	for _, o := range opts {
		o(&options)
	}

	return &cache{
		Registry: r,
		opts:     options,
		running:  make(map[string]bool),
		watched:  make(map[string]watched),
		services: make(map[string]services),
		ttls:     make(map[string]ttls),
		exit:     make(chan bool),
	}
}
