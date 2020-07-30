package client

import (
	"math/rand"

	"github.com/micro/go-micro/v3/errors"
	"github.com/micro/go-micro/v3/router"
	"github.com/micro/go-micro/v3/selector"
)

// LookupRoute for a request using the router and then choose one using the selector
func LookupRoute(req Request, opts CallOptions) (*router.Route, error) {
	// check to see if an address was provided as a call option
	if len(opts.Address) > 0 {
		return &router.Route{
			Service: req.Service(),
			Address: opts.Address[rand.Int()%len(opts.Address)],
		}, nil
	}

	// construct the router query
	query := []router.QueryOption{router.QueryService(req.Service())}

	// if a custom network was requested, pass this to the router. By default the router will use it's
	// own network, which is set during initialisation.
	if len(opts.Network) > 0 {
		query = append(query, router.QueryNetwork(opts.Network))
	}

	// lookup the routes which can be used to execute the request
	routes, err := opts.Router.Lookup(query...)
	if err == router.ErrRouteNotFound {
		return nil, errors.InternalServerError("go.micro.client", "service %s: %s", req.Service(), err.Error())
	} else if err != nil {
		return nil, errors.InternalServerError("go.micro.client", "error getting next %s node: %s", req.Service(), err.Error())
	}

	// select the route to use for the request
	if route, err := opts.Selector.Select(routes, opts.SelectOptions...); err == selector.ErrNoneAvailable {
		return nil, errors.InternalServerError("go.micro.client", "service %s: %s", req.Service(), err.Error())
	} else if err != nil {
		return nil, errors.InternalServerError("go.micro.client", "error getting next %s node: %s", req.Service(), err.Error())
	} else {
		return route, nil
	}
}
