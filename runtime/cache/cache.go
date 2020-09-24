package cache

import (
	"sync"
	"time"

	"github.com/micro/go-micro/v3/runtime"
)

// NewCache wraps a runtime with a cache
func NewCache(r runtime.Runtime) runtime.Runtime {
	return &cache{
		lastUpdated: make(map[string]time.Time),
		services:    make(map[string][]*runtime.Service),
		mux:         new(sync.RWMutex),
		Runtime:     r,
	}
}

type cache struct {
	// lastUpdated contains the last time services were read from the underlying runtime for a given
	// namespace. When a service is Created/Updated/Deleted in a given namespace, the value is deleted
	// from the map
	lastUpdated map[string]time.Time
	// services is a cache of the services in a namespace, it's reset when the lastUpdated is reset
	services map[string][]*runtime.Service
	// mux is a mutex to protect the lastUpdated and services
	mux *sync.RWMutex

	runtime.Runtime
}

// Create a service
func (c *cache) Create(srv *runtime.Service, opts ...runtime.CreateOption) error {
	// parse the options
	var options runtime.CreateOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Namespace == "" {
		options.Namespace = "micro"
	}

	// call the underlying runtime
	if err := c.Runtime.Create(srv, opts...); err != nil {
		return err
	}

	// the service was written so reset the cache for the namespace
	c.mux.Lock()
	delete(c.lastUpdated, options.Namespace)
	delete(c.services, options.Namespace)
	c.mux.Unlock()

	return nil
}

// Read returns the service
func (c *cache) Read(opts ...runtime.ReadOption) ([]*runtime.Service, error) {
	// parse the options
	var options runtime.ReadOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Namespace == "" {
		options.Namespace = "micro"
	}

	// if a query was used we can't use the cache
	if len(options.Service) > 0 || len(options.Type) > 0 || len(options.Version) > 0 {
		return c.Runtime.Read(opts...)
	}

	// check to see if the cache is valid
	c.mux.RLock()
	if t, ok := c.lastUpdated[options.Namespace]; ok && cacheIsValid(t) {
		c.mux.RUnlock()
		return c.services[options.Namespace], nil
	}
	c.mux.RUnlock()

	// the cache was not valid, read from the runtime
	c.mux.Lock()
	defer c.mux.Unlock()

	srvs, err := c.Runtime.Read(opts...)
	if err != nil {
		// if there was an error, don't update the cache
		return nil, err
	}

	c.lastUpdated[options.Namespace] = time.Now()
	c.services[options.Namespace] = srvs
	return srvs, nil
}

// Update the service in place
func (c *cache) Update(srv *runtime.Service, opts ...runtime.UpdateOption) error {
	// parse the options
	var options runtime.UpdateOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Namespace == "" {
		options.Namespace = "micro"
	}

	// call the underlying runtime
	if err := c.Runtime.Update(srv, opts...); err != nil {
		return err
	}

	// the service was updated so reset the cache for the namespace
	c.mux.Lock()
	delete(c.lastUpdated, options.Namespace)
	delete(c.services, options.Namespace)
	c.mux.Unlock()

	return nil
}

// Remove a service
func (c *cache) Delete(srv *runtime.Service, opts ...runtime.DeleteOption) error {
	// parse the options
	var options runtime.DeleteOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Namespace == "" {
		options.Namespace = "micro"
	}

	// call the underlying runtime
	if err := c.Runtime.Delete(srv, opts...); err != nil {
		return err
	}

	// the service was deleted so reset the cache for the namespace
	c.mux.Lock()
	delete(c.lastUpdated, options.Namespace)
	delete(c.services, options.Namespace)
	c.mux.Unlock()

	return nil
}

// String defines the runtime implementation
func (c *cache) String() string {
	return "cache"
}

// cacheIsValid returns a boolean indicating if a cache initialized at the time provided is still
// valid now
func cacheIsValid(t time.Time) bool {
	return t.After(time.Now().Add(time.Second * -30))
}
