package handler

import (
	"context"

	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/network"
	pbNet "github.com/micro/go-micro/network/proto"
	pbRtr "github.com/micro/go-micro/router/proto"
)

// Network implements network handler
type Network struct {
	Network network.Network
}

// ListRoutes returns a list of routing table routes
func (n *Network) ListRoutes(ctx context.Context, req *pbRtr.Request, resp *pbRtr.ListResponse) error {
	routes, err := n.Network.Options().Router.Table().List()
	if err != nil {
		return errors.InternalServerError("go.micro.network", "failed to list routes: %s", err)
	}

	var respRoutes []*pbRtr.Route
	for _, route := range routes {
		respRoute := &pbRtr.Route{
			Service: route.Service,
			Address: route.Address,
			Gateway: route.Gateway,
			Network: route.Network,
			Router:  route.Router,
			Link:    route.Link,
			Metric:  int64(route.Metric),
		}
		respRoutes = append(respRoutes, respRoute)
	}

	resp.Routes = respRoutes

	return nil
}

// ListNodes returns a list of all accessible nodes in the network
func (n *Network) ListNodes(ctx context.Context, req *pbNet.ListRequest, resp *pbNet.ListResponse) error {
	return nil
}

// ListNeighbours returns a list of immediate neighbours
func (n *Network) ListNeighbours(ctx context.Context, req *pbNet.NeighbourhoodRequest, resp *pbNet.NeighbourhoodResponse) error {
	return nil
}
