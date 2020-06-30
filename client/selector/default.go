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
	sopts := SelectOptions{Strategy: c.so.Strategy}
	for _, opt := range opts {
		opt(&sopts)
	}

	// a specific domain was requested, only lookup the services in that domain
	if len(sopts.Domain) > 0 {
		services, err := c.rc.GetService(service, registry.GetDomain(sopts.Domain))
		if err != nil && err != registry.ErrNotFound {
			return nil, err
		}
		for _, filter := range sopts.Filters {
			services = filter(services)
		}
		if len(services) == 0 {
			return nil, ErrNoneAvailable
		}
		return sopts.Strategy(services), nil
	}

	// get the service. Because the service could be running in the current or the default domain,
	// we call both. For example, go.micro.service.foo could be running in the services current domain,
	// however the runtime (go.micro.runtime) will always be run in the default domain.
	services, err := c.rc.GetService(service, registry.GetDomain(c.so.Domain))
	if err != nil && err != registry.ErrNotFound {
		return nil, err
	}

	if c.so.Domain != registry.DefaultDomain {
		srvs, err := c.rc.GetService(service, registry.GetDomain(registry.DefaultDomain))
		if err != nil && err != registry.ErrNotFound {
			return nil, err
		}
		if err == nil {
			services = append(services, srvs...)
		}
	}

	if services == nil {
		return nil, ErrNoneAvailable
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
