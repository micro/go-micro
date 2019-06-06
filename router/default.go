package router

import (
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/gossip"
)

type router struct {
	opts Options
	goss registry.Registry
}

func newRouter(opts ...Option) Router {
	// TODO: for now default options
	goss := gossip.NewRegistry()

	r := &router{
		goss: goss,
	}

	for _, o := range opts {
		o(&r.opts)
	}

	return r
}

func (r *router) Init(opts ...Option) error {
	for _, o := range opts {
		o(&r.opts)
	}
	return nil
}

func (r *router) Options() Options {
	return r.opts
}

func (r *router) AddRoute(route *Route, opts ...RouteOption) error {
	return nil
}

func (r *router) RemoveRoute(route *Route) error {
	return nil
}

func (r *router) GetRoute(s *Service) ([]*Route, error) {
	return nil, nil
}

func (r *router) List() ([]*Route, error) {
	return nil, nil
}

func (r *router) String() string {
	return ""
}
