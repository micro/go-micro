package selector

import (
	"math/rand"
	"time"

	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/selector/internal/blacklist"
)

type defaultSelector struct {
	so   Options
	exit chan bool
	bl   *blacklist.BlackList
}

func init() {
	rand.Seed(time.Now().Unix())
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

	// apply the blacklist
	services, err = r.bl.Filter(services)
	if err != nil {
		return nil, err
	}

	// if there's nothing left, return
	if len(services) == 0 {
		return nil, ErrNoneAvailable
	}

	return sopts.Strategy(services), nil
}

func (r *defaultSelector) Mark(service string, node *registry.Node, err error) {
	r.bl.Mark(service, node, err)
}

func (r *defaultSelector) Reset(service string) {
	r.bl.Reset(service)
}

func (r *defaultSelector) Close() error {
	select {
	case <-r.exit:
		return nil
	default:
		close(r.exit)
		r.bl.Close()
	}
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
		so:   sopts,
		exit: make(chan bool),
		bl:   blacklist.New(),
	}
}
