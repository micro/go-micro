package network

import (
	"container/list"
	"errors"
	"sync"
	"time"

	pb "github.com/micro/go-micro/network/proto"
)

var (
	// MaxDepth defines max depth of peer topology
	MaxDepth uint = 3
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
	// we need to freeze the network graph here
	// otherwise we might get invalid results
	n.RLock()
	defer n.RUnlock()

	//track the visited nodes
	visited := make(map[string]*node)
	// queue of the nodes to visit
	queue := list.New()

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
			peers:   make(map[string]*node),
			network: peer.network,
		}
		// collect peer's peers aka pop (peer of peer)
		for id, pop := range peer.peers {
			p.peers[id] = pop
		}
		peers = append(peers, p)
	}
	n.RUnlock()

	return peers
}

// topology returns network topology up to MaxDepth
func (n *node) Topology(depth uint) *node {
	n.RLock()
	// make a copy of yourself
	node := &node{
		id:      n.id,
		address: n.address,
		peers:   make(map[string]*node),
		network: n.network,
	}

	// return if we reach requested depth or we have no more peers
	if depth == 0 || len(n.peers) == 0 {
		return node
	}

	depth--

	for _, peer := range n.peers {
		nodePeer := peer.Topology(depth)
		node.peers[nodePeer.id] = nodePeer
	}
	n.RUnlock()

	return node
}

// getProtoTopology returns node peers up to given depth encoded in protobufs
// NOTE: this method is NOT thread-safe, so make sure you serialize access to it
func (n *node) getProtoTopology(depth uint) (*pb.Peer, error) {
	node := &pb.Node{
		Id:      n.id,
		Address: n.address,
	}

	pbPeers := &pb.Peer{
		Node:  node,
		Peers: make([]*pb.Peer, 0),
	}

	// return if have either reached the depth or have no more peers
	if depth == 0 || len(n.peers) == 0 {
		return pbPeers, nil
	}

	// decrement the depth
	depth--

	var peers []*pb.Peer
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

// unpackPeer unpacks pb.Peer into node topology of given depth
// NOTE: this method is NOT thread-safe, so make sure you serialize access to it
func unpackPeer(pbPeer *pb.Peer, depth uint) *node {
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
func (n *node) updatePeerTopology(pbPeer *pb.Peer, depth uint) error {
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
