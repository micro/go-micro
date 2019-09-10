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

// ListNodes returns a list of all accessible nodes in the network
func (n *Network) ListNodes(ctx context.Context, req *pbNet.ListRequest, resp *pbNet.ListResponse) error {
	networkNodes := n.Network.Nodes()

	var nodes []*pbNet.Node
	for _, networkNode := range networkNodes {
		node := &pbNet.Node{
			Id:      networkNode.Id(),
			Address: networkNode.Address(),
		}
		nodes = append(nodes, node)
	}

	resp.Nodes = nodes

	return nil
}

// ListPeers returns a list of all the nodes the node has a direct link with
func (n *Network) ListPeers(ctx context.Context, req *pbNet.PeerRequest, resp *pbNet.PeerResponse) error {
	nodePeers := n.Network.Peers()

	var peers []*pbNet.Node
	for _, nodePeer := range nodePeers {
		peer := &pbNet.Node{
			Id:      nodePeer.Id(),
			Address: nodePeer.Address(),
		}
		peers = append(peers, peer)
	}

	resp.Peers = peers

	return nil
}

// Topology returns a list of nodes in node topology i.e. it returns all (in)directly reachable nodes from this node
func (n *Network) Topology(ctx context.Context, req *pbNet.TopologyRequest, resp *pbNet.TopologyResponse) error {
	// NOTE: we are downcasting here
	depth := uint(req.Depth)
	if depth <= 0 {
		depth = network.MaxDepth
	}

	// get topology
	topNodes := n.Network.Topology(depth)

	var nodes []*pbNet.Node
	for _, topNode := range topNodes {
		// creaate peers answer
		pbNode := &pbNet.Node{
			Id:      topNode.Id(),
			Address: topNode.Address(),
		}
		nodes = append(nodes, pbNode)
	}

	// network node
	node := &pbNet.Node{
		Id:      n.Network.Id(),
		Address: n.Network.Address(),
	}

	topology := &pbNet.Topology{
		Node:  node,
		Nodes: nodes,
	}

	resp.Topology = topology

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
