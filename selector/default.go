package selector

import (
	"github.com/micro/go-micro/registry"
)

type defaultSelector struct {
	so Options
}

func (r *defaultSelector) Init(opts ...Option) error {
	for _, o := range opts {
		o(&r.so)
	}
	return nil
}

func (r *defaultSelector) Options() Options {
	return r.so
}

func (r *defaultSelector) Select(service string, opts ...SelectOption) (Next, error) {
	sopts := SelectOptions{
		Strategy: r.so.Strategy,
	}

	for _, opt := range opts {
		opt(&sopts)
	}

	// get the service
	services, err := r.so.Registry.GetService(service)
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

func (r *defaultSelector) Mark(service string, node *registry.Node, err error) {
	return
}

func (r *defaultSelector) Reset(service string) {
	return
}

func (r *defaultSelector) Close() error {
	return nil
}

func (r *defaultSelector) String() string {
	return "default"
}

func newDefaultSelector(opts ...Option) Selector {
	sopts := Options{
		Strategy: Random,
	}

	for _, opt := range opts {
		opt(&sopts)
	}

	if sopts.Registry == nil {
		sopts.Registry = registry.DefaultRegistry
	}

	return &defaultSelector{
		so: sopts,
	}
}
