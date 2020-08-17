package static

import (
	"github.com/micro/go-micro/v3/router"
)

// NewRouter returns an initialized static router
func NewRouter(opts ...router.Option) router.Router {
	options := router.DefaultOptions()
	for _, o := range opts {
		o(&options)
	}
	return &static{options, new(table)}
}

type static struct {
	options router.Options
	table   router.Table
}

func (s *static) Init(opts ...router.Option) error {
	for _, o := range opts {
		o(&s.options)
	}
	return nil
}

func (s *static) Options() router.Options {
	return s.options
}

func (s *static) Table() router.Table {
	return nil
}

func (s *static) Lookup(opts ...router.QueryOption) ([]router.Route, error) {
	return s.table.Query(opts...)
}

func (s *static) Watch(opts ...router.WatchOption) (router.Watcher, error) {
	return nil, nil
}

func (s *static) Close() error {
	return nil
}

func (s *static) String() string {
	return "static"
}

type table struct{}

func (t *table) Create(router.Route) error {
	return nil
}

func (t *table) Delete(router.Route) error {
	return nil
}

func (t *table) Update(router.Route) error {
	return nil
}

func (t *table) List() ([]router.Route, error) {
	return nil, nil
}

func (t *table) Query(opts ...router.QueryOption) ([]router.Route, error) {
	options := router.NewQuery(opts...)

	return []router.Route{
		router.Route{
			Address: options.Service,
			Service: options.Address,
			Gateway: options.Gateway,
			Network: options.Network,
			Router:  options.Router,
		},
	}, nil
}
