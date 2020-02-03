package selector

import (
	"time"

	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/registry/cache"
)

type registrySelector struct {
	so Options
	rc cache.Cache
}

func (c *registrySelector) newCache() cache.Cache {
	ropts := []cache.Option{}
	if c.so.Context != nil {
		if t, ok := c.so.Context.Value("selector_ttl").(time.Duration); ok {
			ropts = append(ropts, cache.WithTTL(t))
		}
	}
	return cache.New(c.so.Registry, ropts...)
}

func (c *registrySelector) Init(opts ...Option) error {
	for _, o := range opts {
		o(&c.so)
	}

	c.rc.Stop()
	c.rc = c.newCache()

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
	services, err := c.rc.GetService(service)
	if err != nil {
		if err == registry.ErrNotFound {
			return nil, ErrNotFound
		}
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
	c.rc.Stop()

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

	s := &registrySelector{
		so: sopts,
	}
	s.rc = s.newCache()

	return s
}
