package client

import (
	"context"
	"sort"

	"github.com/micro/go-micro/v3/errors"
	"github.com/micro/go-micro/v3/router"
)

// LookupFunc is used to lookup routes for a service
type LookupFunc func(context.Context, Request, CallOptions) ([]string, error)

// LookupRoute for a request using the router and then choose one using the selector
func LookupRoute(ctx context.Context, req Request, opts CallOptions) ([]string, error) {
	// check to see if an address was provided as a call option
	if len(opts.Address) > 0 {
		return opts.Address, nil
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

	// sort by lowest metric first
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Metric < routes[j].Metric
	})

	var addrs []string

	for _, route := range routes {
		addrs = append(addrs, route.Address)
	}

	return addrs, nil
}
