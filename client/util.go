package client

import (
	"math/rand"

	"github.com/micro/go-micro/v3/errors"
	"github.com/micro/go-micro/v3/router"
	"github.com/micro/go-micro/v3/selector"
	pnet "github.com/micro/go-micro/v3/util/net"
)

// LookupRoute for a request using the router and then choose one using the selector
func LookupRoute(req Request, opts CallOptions) (*router.Route, error) {
	// check to see if the proxy has been set, if it has we don't need to lookup the routes; net.Proxy
	// returns a slice of addresses, so we'll use a random one. Eventually we should to use the
	// selector for this.
	service, addresses, _ := pnet.Proxy(req.Service(), opts.Address)
	if len(addresses) > 0 {
		return &router.Route{
			Service: service,
			Address: addresses[rand.Int()%len(addresses)],
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
