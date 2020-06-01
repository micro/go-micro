package selector

import (
	"math/rand"

	"github.com/micro/go-micro/v2/router"
)

type random struct{}

func (r *random) Init(opts ...Option) error {
	return nil
}

func (r *random) Options() Options {
	return Options{}
}

func (r *random) Select(routes ...router.Route) (*router.Route, error) {
	// we can't select from an empty pool of routes
	if len(routes) == 0 {
		return nil, ErrNoneAvailable
	}

	// if there is only one route provided we'll select it
	if len(routes) == 1 {
		return &routes[0], nil
	}

	// select a random route from the slice
	return &routes[rand.Intn(len(routes)-1)], nil
}

func (r *random) Record(route *router.Route, err error) error {
	return nil
}

func (r *random) String() string {
	return "random"
}

func newSelector(...Option) Selector {
	return &random{}
}
