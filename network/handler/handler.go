// Package handler implements network RPC handler
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

// ListPeers returns a list of all the nodes the node has a direct link with
func (n *Network) ListPeers(ctx context.Context, req *pbNet.PeerRequest, resp *pbNet.PeerResponse) error {
	depth := uint(req.Depth)
	if depth <= 0 || depth > network.MaxDepth {
		depth = network.MaxDepth
	}

	// get node peers
	nodePeers := n.Network.Peers()

	// get peers encoded into protobuf
	peers := network.PeersToProto(n.Network, nodePeers, depth)

	resp.Peers = peers

	return nil
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
