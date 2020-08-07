package router

import (
	"github.com/micro/go-micro/v3/registry"
	"github.com/micro/go-micro/v3/router"
)

type apiRouter struct {
	routes []router.Route
	router.Router
}

func (r *apiRouter) Lookup(...router.QueryOption) ([]router.Route, error) {
	return r.routes, nil
}

func (r *apiRouter) String() string {
	return "api"
}

// Router is a hack for API routing
func New(srvs []*registry.Service) router.Router {
	var routes []router.Route

	for _, srv := range srvs {
		for _, n := range srv.Nodes {
			routes = append(routes, router.Route{Address: n.Address, Metadata: n.Metadata})
		}
	}

	return &apiRouter{routes: routes}
}
