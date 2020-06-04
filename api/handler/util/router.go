package util

import (
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/router"
)

// Router is a hack for API routing
func Router(srvs []*registry.Service) router.Router {
	var routes []router.Route

	for _, srv := range srvs {
		for _, n := range srv.Nodes {
			routes = append(routes, router.Route{Address: n.Address, Metadata: n.Metadata})
		}
	}

	return &apiRouter{routes: routes}
}

func (r *apiRouter) Lookup(...router.QueryOption) ([]router.Route, error) {
	return r.routes, nil
}

type apiRouter struct {
	routes []router.Route
	router.Router
}

func (r *apiRouter) String() string {
	return "api"
}
