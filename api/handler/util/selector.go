package util

import (
	"math/rand"

	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/router"
	"github.com/micro/go-micro/v2/selector"
)

// Selector is a hack for selection
func Selector(srvs []*registry.Service) selector.Selector {
	var routes []*router.Route

	for _, srv := range srvs {
		for _, n := range srv.Nodes {
			routes = append(routes, &router.Route{Address: n.Address, Metadata: n.Metadata})
		}
	}

	return &apiSelector{routes}
}

type apiSelector struct {
	routes []*router.Route
}

func (s *apiSelector) Init(...selector.Option) error {
	return nil
}

func (s *apiSelector) Options() selector.Options {
	return selector.Options{}
}

func (s *apiSelector) Select(...router.Route) (*router.Route, error) {
	if len(s.routes) == 0 {
		return nil, selector.ErrNoneAvailable
	}
	if len(s.routes) == 1 {
		return s.routes[0], nil
	}
	return s.routes[rand.Intn(len(s.routes)-1)], nil
}

func (s *apiSelector) Record(*router.Route, error) error {
	return nil
}

func (s *apiSelector) String() string {
	return "api"
}
