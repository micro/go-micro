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

// AddPeer adds a new peer to node topology
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

// DeletePeer deletes a peer from node peers
// It returns true if the peers has been deleted
func (n *node) DeletePeer(id string) bool {
	n.Lock()
	defer n.Unlock()

	delete(n.peers, id)

	return true
}

// HasPeer returns true if node has peer with given id
func (n *node) HasPeer(id string) bool {
	n.RLock()
	defer n.RUnlock()

	_, ok := n.peers[id]
	return ok
}

// RefreshPeer updates node timestamp
// It returns false if the peer has not been found.
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

// walk walks the node graph until some condition is met
func (n *node) walk(until func(peer *node) bool) map[string]*node {
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
			if until(node) {
				return visited
			}
		}
		// remove the node from the queue
		queue.Remove(qnode)
	}

	return visited
}

// Nodes returns a slice of all nodes in the whole node topology
func (n *node) Nodes() []Node {
	// we need to freeze the network graph here
	// otherwise we might get inconsisten results
	n.RLock()
	defer n.RUnlock()

	// NOTE: this should never be true
	untilNoMorePeers := func(n *node) bool {
		return n == nil
	}

	visited := n.walk(untilNoMorePeers)

	var nodes []Node
	// collect all the nodes and return them
	for _, node := range visited {
		nodes = append(nodes, node)
	}

	return nodes
}

// GetPeerNode returns a node from node MaxDepth topology
// It returns nil if the peer was not found
func (n *node) GetPeerNode(id string) *node {
	n.RLock()
	defer n.RUnlock()

	// get node topology up to MaxDepth
	top := n.Topology(MaxDepth)

	untilFoundPeer := func(n *node) bool {
		return n.id == id
	}

	visited := top.walk(untilFoundPeer)

	peerNode, ok := visited[id]
	if !ok {
		return nil
	}

	return peerNode
}

// Topology returns a copy of the node topology down to given depth
// NOTE: the returned node is a node graph - not a single node
func (n *node) Topology(depth uint) *node {
	n.RLock()
	defer n.RUnlock()

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
		nodePeer := peer.Topology(depth)
		if _, ok := node.peers[nodePeer.id]; !ok {
			node.peers[nodePeer.id] = nodePeer
		}
	}

	return node
}

// Peers returns node peers up to MaxDepth
func (n *node) Peers() []Node {
	n.RLock()
	defer n.RUnlock()

	var peers []Node
	for _, nodePeer := range n.peers {
		peer := nodePeer.Topology(MaxDepth)
		peers = append(peers, peer)
	}

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

func peerProtoTopology(peer Node, depth uint) *pb.Peer {
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
		peer := peerProtoTopology(pop, depth)
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
		pbPeer := peerProtoTopology(peer, depth)
		pbPeers.Peers = append(pbPeers.Peers, pbPeer)
	}

	return pbPeers
}
