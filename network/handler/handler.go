// Package handler implements network RPC handler
package handler

import (
	"context"
	"sort"

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
	nodes := n.Network.Nodes()

	var respNodes []*pbNet.Node
	for _, node := range nodes {
		respNode := &pbNet.Node{
			Id:      node.Id(),
			Address: node.Address(),
		}
		respNodes = append(respNodes, respNode)
	}

	resp.Nodes = respNodes

	return nil
}

// ListPeers returns a list of all the nodes the node has a direct link with
func (n *Network) ListPeers(ctx context.Context, req *pbNet.PeerRequest, resp *pbNet.PeerResponse) error {
	// extract the id of the node to query
	id := req.Id
	// if no id is passed, we assume local node
	if id == "" {
		id = n.Network.Id()
	}

	// get all the nodes in the network
	nodes := n.Network.Nodes()

	// sort the slice of nodes
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Id() <= nodes[j].Id() })
	// find a node with a given id
	i := sort.Search(len(nodes), func(j int) bool { return nodes[j].Id() >= id })

	var nodePeers []*pbNet.Node
	// collect all the node peers into slice
	if i < len(nodes) && nodes[i].Id() == id {
		for _, peer := range nodes[i].Peers() {
			// don't return yourself in response
			if peer.Id() == n.Network.Id() {
				continue
			}
			pbPeer := &pbNet.Node{
				Id:      peer.Id(),
				Address: peer.Address(),
			}
			nodePeers = append(nodePeers, pbPeer)
		}
	}

	// requested node
	node := &pbNet.Node{
		Id:      nodes[i].Id(),
		Address: nodes[i].Address(),
	}

	// creaate peers answer
	peers := &pbNet.Peers{
		Node:  node,
		Peers: nodePeers,
	}

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
