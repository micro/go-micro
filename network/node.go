package network

import (
	"container/list"
	"errors"
	"sync"
	"time"

	pb "github.com/micro/go-micro/v2/network/service/proto"
)

var (
	// MaxDepth defines max depth of peer topology
	MaxDepth uint = 3
)

var (
	// ErrPeerExists is returned when adding a peer which already exists
	ErrPeerExists = errors.New("peer already exists")
	// ErrPeerNotFound is returned when a peer could not be found in node topology
	ErrPeerNotFound = errors.New("peer not found")
)

// nodeError tracks node errors
type nodeError struct {
	sync.RWMutex
	count int
	msg   error
}

// Increment increments node error count
func (e *nodeError) Update(err error) {
	e.Lock()
	defer e.Unlock()

	e.count++
	e.msg = err
}

// Count returns node error count
func (e *nodeError) Count() int {
	e.RLock()
	defer e.RUnlock()

	return e.count
}

func (e *nodeError) Msg() string {
	e.RLock()
	defer e.RUnlock()

	if e.msg != nil {
		return e.msg.Error()
	}

	return ""
}

// status returns node status
type status struct {
	sync.RWMutex
	err *nodeError
}

// newStatus creates
func newStatus() *status {
	return &status{
		err: new(nodeError),
	}
}

func newPeerStatus(peer *pb.Peer) *status {
	status := &status{
		err: new(nodeError),
	}

	// if Node.Status is nil, return empty status
	if peer.Node.Status == nil {
		return status
	}

	// if peer.Node.Status.Error is NOT nil, update status fields
	if err := peer.Node.Status.GetError(); err != nil {
		status.err.count = int(peer.Node.Status.Error.Count)
		status.err.msg = errors.New(peer.Node.Status.Error.Msg)
	}

	return status
}

func (s *status) Error() Error {
	s.RLock()
	defer s.RUnlock()

	return &nodeError{
		count: s.err.count,
		msg:   s.err.msg,
	}
}

// node is network node
type node struct {
	sync.RWMutex
	// id is node id
	id string
	// address is node address
	address string
	// link on which we communicate with the peer
	link string
	// peers are nodes with direct link to this node
	peers map[string]*node
	// network returns the node network
	network Network
	// lastSeen keeps track of node lifetime and updates
	lastSeen time.Time
	// lastSync keeps track of node last sync request
	lastSync time.Time
	// err tracks node status
	status *status
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

// Status returns node status
func (n *node) Status() Status {
	n.RLock()
	defer n.RUnlock()

	return &status{
		err: &nodeError{
			count: n.status.err.count,
			msg:   n.status.err.msg,
		},
	}
}

// walk walks the node graph until some condition is met
func (n *node) walk(until func(peer *node) bool, action func(parent, peer *node)) map[string]*node {
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
		if until(qnode.Value.(*node)) {
			return visited
		}
		// iterate through all of the node peers
		// mark the visited nodes; enqueue the non-visted
		for id, peer := range qnode.Value.(*node).peers {
			action(qnode.Value.(*node), peer)
			if _, ok := visited[id]; !ok {
				visited[id] = peer
				queue.PushBack(peer)
			}
		}
		// remove the node from the queue
		queue.Remove(qnode)
	}

	return visited
}

// AddPeer adds a new peer to node topology
// It returns false if the peer already exists
func (n *node) AddPeer(peer *node) error {
	n.Lock()
	defer n.Unlock()

	// get node topology: we need to check if the peer
	// we are trying to add is already in our graph
	top := n.getTopology(MaxDepth)

	untilFoundPeer := func(n *node) bool {
		return n.id == peer.id
	}

	justWalk := func(paent, node *node) {}

	visited := top.walk(untilFoundPeer, justWalk)

	peerNode, inTop := visited[peer.id]

	if _, ok := n.peers[peer.id]; !ok {
		if inTop {
			// just create a new edge to the existing peer
			// but make sure you update the peer link
			peerNode.link = peer.link
			n.peers[peer.id] = peerNode
			return nil
		}
		n.peers[peer.id] = peer
		return nil
	}

	return ErrPeerExists
}

// DeletePeer deletes a peer from node peers
// It returns true if the peer has been deleted
func (n *node) DeletePeer(id string) bool {
	n.Lock()
	defer n.Unlock()

	delete(n.peers, id)

	return true
}

// UpdatePeer updates a peer if it already exists
// It returns error if the peer does not exist
func (n *node) UpdatePeer(peer *node) error {
	n.Lock()
	defer n.Unlock()

	if _, ok := n.peers[peer.id]; ok {
		n.peers[peer.id] = peer
		return nil
	}

	return ErrPeerNotFound
}

// RefreshPeer updates node last seen timestamp
// It returns false if the peer has not been found.
func (n *node) RefreshPeer(id, link string, now time.Time) error {
	n.Lock()
	defer n.Unlock()

	peer, ok := n.peers[id]
	if !ok {
		return ErrPeerNotFound
	}

	// set peer link
	peer.link = link
	// set last seen
	peer.lastSeen = now

	return nil
}

// RefreshSync refreshes nodes sync time
func (n *node) RefreshSync(now time.Time) error {
	n.Lock()
	defer n.Unlock()

	n.lastSync = now

	return nil
}

// Nodes returns a slice of all nodes in the whole node topology
func (n *node) Nodes() []Node {
	// we need to freeze the network graph here
	// otherwise we might get inconsisten results
	n.RLock()
	defer n.RUnlock()

	// NOTE: this should never be true
	untilNoMorePeers := func(node *node) bool {
		return node == nil
	}
	justWalk := func(parent, node *node) {}

	visited := n.walk(untilNoMorePeers, justWalk)

	nodes := make([]Node, 0, len(visited))
	// collect all the nodes and return them
	for _, node := range visited {
		nodes = append(nodes, node)
	}

	return nodes
}

// GetPeerNode returns a node from node MaxDepth topology
// It returns nil if the peer was not found
func (n *node) GetPeerNode(id string) *node {
	// get node topology up to MaxDepth
	top := n.Topology(MaxDepth)

	untilFoundPeer := func(n *node) bool {
		return n.id == id
	}
	justWalk := func(paent, node *node) {}

	visited := top.walk(untilFoundPeer, justWalk)

	peerNode, ok := visited[id]
	if !ok {
		return nil
	}

	return peerNode
}

// DeletePeerNode removes peer node from node topology
func (n *node) DeletePeerNode(id string) error {
	n.Lock()
	defer n.Unlock()

	untilNoMorePeers := func(node *node) bool {
		return node == nil
	}

	deleted := make(map[string]*node)
	deletePeer := func(parent, node *node) {
		if node.id != n.id && node.id == id {
			delete(parent.peers, node.id)
			deleted[node.id] = node
		}
	}

	n.walk(untilNoMorePeers, deletePeer)

	if _, ok := deleted[id]; !ok {
		return ErrPeerNotFound
	}

	return nil
}

// PrunePeer prunes the peers with the given id
func (n *node) PrunePeer(id string) {
	n.Lock()
	defer n.Unlock()

	untilNoMorePeers := func(node *node) bool {
		return node == nil
	}

	prunePeer := func(parent, node *node) {
		if node.id != n.id && node.id == id {
			delete(parent.peers, node.id)
		}
	}

	n.walk(untilNoMorePeers, prunePeer)
}

// PruneStalePeerNodes prunes the peers that have not been seen for longer than pruneTime
// It returns a map of the the nodes that got pruned
func (n *node) PruneStalePeers(pruneTime time.Duration) map[string]*node {
	n.Lock()
	defer n.Unlock()

	untilNoMorePeers := func(node *node) bool {
		return node == nil
	}

	pruned := make(map[string]*node)
	pruneStalePeer := func(parent, node *node) {
		if node.id != n.id && time.Since(node.lastSeen) > PruneTime {
			delete(parent.peers, node.id)
			pruned[node.id] = node
		}
	}

	n.walk(untilNoMorePeers, pruneStalePeer)

	return pruned
}

// getTopology traverses node graph and builds node topology
// NOTE: this function is not thread safe
func (n *node) getTopology(depth uint) *node {
	// make a copy of yourself
	node := &node{
		id:       n.id,
		address:  n.address,
		peers:    make(map[string]*node),
		network:  n.network,
		status:   n.status,
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
		nodePeer := peer.getTopology(depth)
		if _, ok := node.peers[nodePeer.id]; !ok {
			node.peers[nodePeer.id] = nodePeer
		}
	}

	return node
}

// Topology returns a copy of the node topology down to given depth
// NOTE: the returned node is a node graph - not a single node
func (n *node) Topology(depth uint) *node {
	n.RLock()
	defer n.RUnlock()

	return n.getTopology(depth)
}

// Peers returns node peers up to MaxDepth
func (n *node) Peers() []Node {
	n.RLock()
	defer n.RUnlock()

	peers := make([]Node, 0, len(n.peers))
	for _, nodePeer := range n.peers {
		peer := nodePeer.getTopology(MaxDepth)
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
		status:   newPeerStatus(pbPeer),
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
		Status: &pb.Status{
			Error: &pb.Error{
				Count: uint32(peer.Status().Error().Count()),
				Msg:   peer.Status().Error().Msg(),
			},
		},
	}

	// set the network name if network is not nil
	if peer.Network() != nil {
		node.Network = peer.Network().Name()
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
		Status: &pb.Status{
			Error: &pb.Error{
				Count: uint32(node.Status().Error().Count()),
				Msg:   node.Status().Error().Msg(),
			},
		},
	}

	// set the network name if network is not nil
	if node.Network() != nil {
		pbNode.Network = node.Network().Name()
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
