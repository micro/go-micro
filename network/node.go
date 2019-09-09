package network

import (
	"errors"
	"sync"
	"time"

	pbNet "github.com/micro/go-micro/network/proto"
)

var (
	// MaxDepth defines max depth of peer topology
	MaxDepth = 3
)

// node is network node
type node struct {
	sync.RWMutex
	// id is node id
	id string
	// address is node address
	address string
	// neighbours maps the node neighbourhood
	neighbours map[string]*node
	// network returns the node network
	network Network
	// lastSeen stores the time the node has been seen last time
	lastSeen time.Time
}

// Id is node ide
func (n *node) Id() string {
	return n.id
}

// Address returns node address
func (n *node) Address() string {
	return n.address
}

// Network returns node network
func (n *node) Network() Network {
	return n.network
}

// Neighbourhood returns node neighbourhood
func (n *node) Neighbourhood() []Node {
	var nodes []Node
	n.RLock()
	for _, neighbourNode := range n.neighbours {
		// make a copy of the node
		n := &node{
			id:      neighbourNode.id,
			address: neighbourNode.address,
			network: neighbourNode.network,
		}
		// NOTE: we do not care about neighbour's neighbours
		nodes = append(nodes, n)
	}
	n.RUnlock()

	return nodes
}

// getNeighbours collects node neighbours up to given depth into pbNeighbours
// NOTE: this method is not thread safe, so make sure you serialize access to it
// NOTE: we should be able to read-Lock this, even though it's recursive
func (n *node) getNeighbours(depth int) (*pbNet.Neighbour, error) {
	node := &pbNet.Node{
		Id:      n.id,
		Address: n.address,
	}
	pbNeighbours := &pbNet.Neighbour{
		Node:       node,
		Neighbours: make([]*pbNet.Neighbour, 0),
	}

	// return if have either reached the depth or have no more neighbours
	if depth == 0 || len(n.neighbours) == 0 {
		return pbNeighbours, nil
	}

	// decrement the depth
	depth--

	var neighbours []*pbNet.Neighbour
	for _, neighbour := range n.neighbours {
		// get neighbours of the neighbour
		// NOTE: this is [not] a recursive call
		pbNodeNeighbour, err := neighbour.getNeighbours(depth)
		if err != nil {
			return nil, err
		}
		// add current neighbour to explored neighbours
		neighbours = append(neighbours, pbNodeNeighbour)
	}

	// add neighbours to the parent topology
	pbNeighbours.Neighbours = neighbours

	return pbNeighbours, nil
}

// unpackNeighbour unpacks pbNet.Neighbour into node of given depth
// NOTE: this method is not thread safe, so make sure you serialize access to it
func unpackNeighbour(pbNeighbour *pbNet.Neighbour, depth int) (*node, error) {
	if pbNeighbour == nil {
		return nil, errors.New("neighbour not initialized")
	}

	neighbourNode := &node{
		id:         pbNeighbour.Node.Id,
		address:    pbNeighbour.Node.Address,
		neighbours: make(map[string]*node),
	}

	// return if have either reached the depth or have no more neighbours
	if depth == 0 || len(pbNeighbour.Neighbours) == 0 {
		return neighbourNode, nil
	}

	// decrement the depth
	depth--

	neighbours := make(map[string]*node)
	for _, pbNode := range pbNeighbour.Neighbours {
		node, err := unpackNeighbour(pbNode, depth)
		if err != nil {
			return nil, err
		}
		neighbours[pbNode.Node.Id] = node
	}

	neighbourNode.neighbours = neighbours

	return neighbourNode, nil
}

// updateNeighbour updates node neighbour up to given depth
// NOTE: this method is not thread safe, so make sure you serialize access to it
func (n *node) updateNeighbour(neighbour *pbNet.Neighbour, depth int) error {
	// unpack neighbour into topology of size MaxDepth-1
	// NOTE: we need MaxDepth-1 because node n is the parent adding which
	// gives us the max neighbour topology we maintain and propagate
	node, err := unpackNeighbour(neighbour, MaxDepth-1)
	if err != nil {
		return err
	}

	// update node neighbours with new topology
	n.neighbours[neighbour.Node.Id] = node

	return nil
}
