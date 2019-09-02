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

// Neighbourhood returns a list of immediate neighbours
func (n *Network) Neighbourhood(ctx context.Context, req *pbNet.NeighbourhoodRequest, resp *pbNet.NeighbourhoodResponse) error {
	// extract the id of the node to query
	id := req.Id
	// if no id is passed, we assume local node
	if id == "" {
		id = n.Network.Id()
	}

	// get all the nodes in the network
	nodes := n.Network.Nodes()

	var neighbours []*pbNet.Neighbour
	// find a node with a given id
	i := sort.Search(len(nodes), func(i int) bool { return nodes[i].Id() == id })
	// collect all the nodes in the neighbourhood of the found node
	if i < len(nodes) && nodes[i].Id() == id {
		for _, neighbour := range nodes[i].Neighbourhood() {
			var nodeNeighbours []*pbNet.Node
			for _, nodeNeighbour := range neighbour.Neighbourhood() {
				nn := &pbNet.Node{
					Id:      nodeNeighbour.Id(),
					Address: nodeNeighbour.Address(),
				}
				nodeNeighbours = append(nodeNeighbours, nn)
			}
			// node is present at node[i]
			neighbour := &pbNet.Neighbour{
				Node: &pbNet.Node{
					Id:      neighbour.Id(),
					Address: neighbour.Address(),
				},
				Neighbours: nodeNeighbours,
			}
			neighbours = append(neighbours, neighbour)
		}
	}

	resp.Neighbours = neighbours

	return nil
}
