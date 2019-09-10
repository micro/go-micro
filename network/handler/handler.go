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

// toplogyToProto recursively traverses node topology and returns it
func toplogyToProto(node network.Node, pbPeer *pbNet.Peer) *pbNet.Peer {
	// return if we reached the end of topology
	if len(node.Peers()) == 0 {
		return pbPeer
	}

	for _, topNode := range node.Peers() {
		pbNode := &pbNet.Node{
			Id:      topNode.Id(),
			Address: topNode.Address(),
		}
		pbPeer := &pbNet.Peer{
			Node:  pbNode,
			Peers: make([]*pbNet.Peer, 0),
		}
		peer := toplogyToProto(topNode, pbPeer)
		pbPeer.Peers = append(pbPeer.Peers, peer)
	}

	return pbPeer
}

// Topology returns a list of nodes in node topology i.e. it returns all (in)directly reachable nodes from this node
func (n *Network) Topology(ctx context.Context, req *pbNet.TopologyRequest, resp *pbNet.TopologyResponse) error {
	// get node topology
	topNode := n.Network.Topology()

	// network node aka root node
	node := &pbNet.Node{
		Id:      n.Network.Id(),
		Address: n.Network.Address(),
	}
	// we will build proto topology into this
	pbPeer := &pbNet.Peer{
		Node:  node,
		Peers: make([]*pbNet.Peer, 0),
	}
	// return topology encoded into protobuf
	topology := toplogyToProto(topNode, pbPeer)

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
