package network

import (
	"container/list"
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

// AddPeer adds a new peer to node
// It returns false if the peer already exists
func (n *node) AddPeer(peer *node) bool {
	n.Lock()
	defer n.Unlock()

	if _, ok := n.peers[peer.id]; !ok {
		n.peers[peer.id] = peer
		return true
	}

	return false
}

// GetPeer returns a peer if it exists
// It returns nil if the peer was not found
func (n *node) GetPeer(id string) *node {
	n.RLock()
	defer n.RUnlock()

	p, ok := n.peers[id]
	if !ok {
		return nil
	}

	peer := &node{
		id:       p.id,
		address:  p.address,
		peers:    make(map[string]*node),
		network:  p.network,
		lastSeen: p.lastSeen,
	}

	// TODO: recursively retrieve all of its peers
	for id, pop := range p.peers {
		peer.peers[id] = pop
	}

	return peer
}

// UpdatePeer updates a peer if it exists
// It returns false if the peer does not exist
func (n *node) UpdatePeer(peer *node) bool {
	n.Lock()
	defer n.Unlock()

	if _, ok := n.peers[peer.id]; ok {
		n.peers[peer.id] = peer
		return true
	}

	return false
}

// DeletePeer deletes a peer if it exists
func (n *node) DeletePeer(id string) {
	n.Lock()
	defer n.Unlock()

	delete(n.peers, id)
}

// Refresh updates node timestamp
func (n *node) RefreshPeer(id string, now time.Time) bool {
	n.Lock()
	defer n.Unlock()

	peer, ok := n.peers[id]
	if !ok {
		return false
	}

	if peer.lastSeen.Before(now) {
		peer.lastSeen = now
	}

	return true
}

// Nodes returns a slice if all nodes in node topology
func (n *node) Nodes() []Node {
	// we need to freeze the network graph here
	// otherwise we might get inconsisten results
	n.RLock()
	defer n.RUnlock()

	// track the visited nodes
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

// topology returns node topology down to given depth
func (n *node) topology(depth uint) *node {
	// make a copy of yourself
	node := &node{
		id:       n.id,
		address:  n.address,
		peers:    make(map[string]*node),
		network:  n.network,
		lastSeen: n.lastSeen,
	}

	// return if we reach requested depth or we have no more peers
	if depth == 0 || len(n.peers) == 0 {
		return node
	}

	// decrement the depth
	depth--

	// iterate through our peers and update the node peers
	for _, peer := range n.peers {
		nodePeer := peer.topology(depth)
		if _, ok := node.peers[nodePeer.id]; !ok {
			node.peers[nodePeer.id] = nodePeer
		}
	}

	return node
}

// Peers returns node peers
func (n *node) Peers() []Node {
	n.RLock()
	var peers []Node
	for _, nodePeer := range n.peers {
		peer := nodePeer.topology(MaxDepth)
		peers = append(peers, peer)
	}
	n.RUnlock()

	return peers
}

// UnpackPeerTopology unpacks pb.Peer into node topology of given depth
func UnpackPeerTopology(pbPeer *pb.Peer, lastSeen time.Time, depth uint) *node {
	peerNode := &node{
		id:       pbPeer.Node.Id,
		address:  pbPeer.Node.Address,
		peers:    make(map[string]*node),
		lastSeen: lastSeen,
	}

	// return if have either reached the depth or have no more peers
	if depth == 0 || len(pbPeer.Peers) == 0 {
		return peerNode
	}

	// decrement the depth
	depth--

	peers := make(map[string]*node)
	for _, pbPeer := range pbPeer.Peers {
		peer := UnpackPeerTopology(pbPeer, lastSeen, depth)
		peers[pbPeer.Node.Id] = peer
	}

	peerNode.peers = peers

	return peerNode
}

func peerTopology(peer Node, depth uint) *pb.Peer {
	node := &pb.Node{
		Id:      peer.Id(),
		Address: peer.Address(),
	}

	pbPeers := &pb.Peer{
		Node:  node,
		Peers: make([]*pb.Peer, 0),
	}

	// return if we reached the end of topology or depth
	if depth == 0 || len(peer.Peers()) == 0 {
		return pbPeers
	}

	// decrement the depth
	depth--

	// iterate through peers of peers aka pops
	for _, pop := range peer.Peers() {
		peer := peerTopology(pop, depth)
		pbPeers.Peers = append(pbPeers.Peers, peer)
	}

	return pbPeers
}

// PeersToProto returns node peers graph encoded into protobuf
func PeersToProto(node Node, depth uint) *pb.Peer {
	// network node aka root node
	pbNode := &pb.Node{
		Id:      node.Id(),
		Address: node.Address(),
	}
	// we will build proto topology into this
	pbPeers := &pb.Peer{
		Node:  pbNode,
		Peers: make([]*pb.Peer, 0),
	}

	for _, peer := range node.Peers() {
		pbPeer := peerTopology(peer, depth)
		pbPeers.Peers = append(pbPeers.Peers, pbPeer)
	}

	return pbPeers
}
