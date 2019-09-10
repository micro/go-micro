package network

import (
	"container/list"
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
	// peers are nodes with direct link to this node
	peers map[string]*node
	// network returns the node network
	network Network
	// lastSeen keeps track of node lifetime and updates
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

// Nodes returns a slice if all nodes in node topology
func (n *node) Nodes() []Node {
	//track the visited nodes
	visited := make(map[string]*node)
	// queue of the nodes to visit
	queue := list.New()

	// we need to freeze the network graph here
	// otherwise we might get invalid results
	n.RLock()
	defer n.RUnlock()

	// push node to the back of queue
	queue.PushBack(n)
	// mark the node as visited
	visited[n.id] = n

	// keep iterating over the queue until its empty
	for queue.Len() > 0 {
		// pop the node from the front of the queue
		qnode := queue.Front()
		// iterate through all of the node peers
		// mark the visited nodes; enqueue the non-visted
		for id, node := range qnode.Value.(*node).peers {
			if _, ok := visited[id]; !ok {
				visited[id] = node
				queue.PushBack(node)
			}
		}
		// remove the node from the queue
		queue.Remove(qnode)
	}

	var nodes []Node
	// collect all the nodes and return them
	for _, node := range visited {
		nodes = append(nodes, node)
	}

	return nodes
}

// Peers returns node peers
func (n *node) Peers() []Node {
	var peers []Node
	n.RLock()
	for _, peer := range n.peers {
		// make a copy of the node
		p := &node{
			id:      peer.id,
			address: peer.address,
			network: peer.network,
		}
		// NOTE: we do not care about peer's peers
		peers = append(peers, p)
	}
	n.RUnlock()

	return peers
}

// getProtoTopology returns node peers up to given depth encoded in protobufs
// NOTE: this method is NOT thread-safe, so make sure you serialize access to it
func (n *node) getProtoTopology(depth int) (*pbNet.Peer, error) {
	node := &pbNet.Node{
		Id:      n.id,
		Address: n.address,
	}

	pbPeers := &pbNet.Peer{
		Node:  node,
		Peers: make([]*pbNet.Peer, 0),
	}

	// return if have either reached the depth or have no more peers
	if depth == 0 || len(n.peers) == 0 {
		return pbPeers, nil
	}

	// decrement the depth
	depth--

	var peers []*pbNet.Peer
	for _, peer := range n.peers {
		// get peers of the node peers
		// NOTE: this is [not] a recursive call
		pbPeerPeer, err := peer.getProtoTopology(depth)
		if err != nil {
			return nil, err
		}
		// add current peer to explored peers
		peers = append(peers, pbPeerPeer)
	}

	// add peers to the parent topology
	pbPeers.Peers = peers

	return pbPeers, nil
}

// unpackPeer unpacks pbNet.Peer into node topology of given depth
// NOTE: this method is NOT thread-safe, so make sure you serialize access to it
func unpackPeer(pbPeer *pbNet.Peer, depth int) *node {
	peerNode := &node{
		id:      pbPeer.Node.Id,
		address: pbPeer.Node.Address,
		peers:   make(map[string]*node),
	}

	// return if have either reached the depth or have no more peers
	if depth == 0 || len(pbPeer.Peers) == 0 {
		return peerNode
	}

	// decrement the depth
	depth--

	peers := make(map[string]*node)
	for _, pbPeer := range pbPeer.Peers {
		peer := unpackPeer(pbPeer, depth)
		peers[pbPeer.Node.Id] = peer
	}

	peerNode.peers = peers

	return peerNode
}

// updatePeer updates node peer up to given depth
// NOTE: this method is not thread safe, so make sure you serialize access to it
func (n *node) updatePeerTopology(pbPeer *pbNet.Peer, depth int) error {
	if pbPeer == nil {
		return errors.New("peer not initialized")
	}

	// NOTE: we need MaxDepth-1 because node n is the parent adding which
	// gives us the max peer topology we maintain and propagate
	peer := unpackPeer(pbPeer, MaxDepth-1)

	// update node peers with new topology
	n.peers[pbPeer.Node.Id] = peer

	return nil
}
