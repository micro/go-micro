package network

import (
	"testing"
	"time"

	pb "github.com/micro/go-micro/v2/network/service/proto"
)

var (
	testNodeId        = "testNode"
	testNodeAddress   = "testAddress"
	testNodeNetName   = "testNetwork"
	testNodePeerIds   = []string{"peer1", "peer2", "peer3"}
	testPeerOfPeerIds = []string{"peer11", "peer12"}
)

func testSetup() *node {
	testNode := &node{
		id:      testNodeId,
		address: testNodeAddress,
		peers:   make(map[string]*node),
		network: newNetwork(Name(testNodeNetName)),
		status:  newStatus(),
	}

	// add some peers to the node
	for _, id := range testNodePeerIds {
		testNode.peers[id] = &node{
			id:      id,
			address: testNode.address + "-" + id,
			peers:   make(map[string]*node),
			network: testNode.network,
			status:  newStatus(),
		}
	}

	// add peers to peer1
	// NOTE: these are peers of peers!
	for _, id := range testPeerOfPeerIds {
		testNode.peers["peer1"].peers[id] = &node{
			id:      id,
			address: testNode.address + "-" + id,
			peers:   make(map[string]*node),
			network: testNode.network,
			status:  newStatus(),
		}
	}

	// connect peer1 with peer2
	testNode.peers["peer1"].peers["peer2"] = testNode.peers["peer2"]
	// connect peer2 with peer3
	testNode.peers["peer2"].peers["peer3"] = testNode.peers["peer3"]

	return testNode
}

func TestNodeId(t *testing.T) {
	node := testSetup()
	if node.Id() != testNodeId {
		t.Errorf("Expected id: %s, found: %s", testNodeId, node.Id())
	}
}

func TestNodeAddress(t *testing.T) {
	node := testSetup()
	if node.Address() != testNodeAddress {
		t.Errorf("Expected address: %s, found: %s", testNodeAddress, node.Address())
	}
}
func TestNodeNetwork(t *testing.T) {
	node := testSetup()
	if node.Network().Name() != testNodeNetName {
		t.Errorf("Expected network: %s, found: %s", testNodeNetName, node.Network().Name())
	}
}

func TestNodes(t *testing.T) {
	// single node
	single := &node{
		id:      testNodeId,
		address: testNodeAddress,
		peers:   make(map[string]*node),
		network: newNetwork(Name(testNodeNetName)),
	}
	// get all the nodes including yourself
	nodes := single.Nodes()
	nodeCount := 1

	if len(nodes) != nodeCount {
		t.Errorf("Expected to find %d nodes, found: %d", nodeCount, len(nodes))
	}

	// complicated node graph
	node := testSetup()
	// get all the nodes including yourself
	nodes = node.Nodes()

	// compile a list of ids of all nodes in the network into map for easy indexing
	nodeIds := make(map[string]bool)
	// add yourself
	nodeIds[node.id] = true
	// add peer Ids
	for _, id := range testNodePeerIds {
		nodeIds[id] = true
	}
	// add peer1 peers i.e. peers of peer
	for _, id := range testPeerOfPeerIds {
		nodeIds[id] = true
	}

	// we should return the correct number of nodes
	if len(nodes) != len(nodeIds) {
		t.Errorf("Expected %d nodes, found: %d", len(nodeIds), len(nodes))
	}

	// iterate through the list of nodes and makes sure all have been returned
	for _, node := range nodes {
		if _, ok := nodeIds[node.Id()]; !ok {
			t.Errorf("Expected to find %s node", node.Id())
		}
	}

	// this is a leaf node
	id := "peer11"
	if nodePeer := node.GetPeerNode(id); nodePeer == nil {
		t.Errorf("Expected to find %s node", id)
	}
}

func collectPeerIds(peer Node, ids map[string]bool) map[string]bool {
	if len(peer.Peers()) == 0 {
		return ids
	}

	// iterate through the whole graph
	for _, peer := range peer.Peers() {
		ids = collectPeerIds(peer, ids)
		if _, ok := ids[peer.Id()]; !ok {
			ids[peer.Id()] = true
		}
	}

	return ids
}

func TestPeers(t *testing.T) {
	// single node
	single := &node{
		id:      testNodeId,
		address: testNodeAddress,
		peers:   make(map[string]*node),
		network: newNetwork(Name(testNodeNetName)),
	}
	// get node peers
	peers := single.Peers()
	// there should be no peers
	peerCount := 0

	if len(peers) != peerCount {
		t.Errorf("Expected to find %d nodes, found: %d", peerCount, len(peers))
	}

	// complicated node graph
	node := testSetup()
	// list of ids of nodes of MaxDepth
	peerIds := make(map[string]bool)
	// add peer Ids
	for _, id := range testNodePeerIds {
		peerIds[id] = true
	}
	// add peers of peers to peerIds
	for _, id := range testPeerOfPeerIds {
		peerIds[id] = true
	}
	// get node peers
	peers = node.Peers()

	// we will collect all returned Peer Ids into this map
	resPeerIds := make(map[string]bool)
	for _, peer := range peers {
		resPeerIds[peer.Id()] = true
		resPeerIds = collectPeerIds(peer, resPeerIds)
	}

	// if correct, we must collect all peerIds
	if len(resPeerIds) != len(peerIds) {
		t.Errorf("Expected to find %d peers, found: %d", len(peerIds), len(resPeerIds))
	}

	for id := range resPeerIds {
		if _, ok := peerIds[id]; !ok {
			t.Errorf("Expected to find %s peer", id)
		}
	}
}

func TestDeletePeerNode(t *testing.T) {
	// complicated node graph
	node := testSetup()

	nodeCount := len(node.Nodes())

	// should not find non-existent peer node
	if err := node.DeletePeerNode("foobar"); err != ErrPeerNotFound {
		t.Errorf("Expected: %v, got: %v", ErrPeerNotFound, err)
	}

	// lets pick one of the peer1 peers
	if err := node.DeletePeerNode(testPeerOfPeerIds[0]); err != nil {
		t.Errorf("Error deleting peer node: %v", err)
	}

	nodeDelCount := len(node.Nodes())

	if nodeDelCount != nodeCount-1 {
		t.Errorf("Expected node count: %d, got: %d", nodeCount-1, nodeDelCount)
	}
}

func TestPrunePeer(t *testing.T) {
	// complicated node graph
	node := testSetup()

	before := node.Nodes()

	node.PrunePeer("peer3")

	now := node.Nodes()

	if len(now) != len(before)-1 {
		t.Errorf("Expected pruned node count: %d, got: %d", len(before)-1, len(now))
	}
}

func TestPruneStalePeers(t *testing.T) {
	// complicated node graph
	node := testSetup()
	nodes := node.Nodes()
	// this will delete all nodes besides the root node
	pruneTime := 10 * time.Millisecond
	time.Sleep(pruneTime)

	// should delete all nodes besides (root) node
	pruned := node.PruneStalePeers(pruneTime)

	if len(pruned) != len(nodes)-1 {
		t.Errorf("Expected pruned node count: %d, got: %d", len(nodes)-1, len(pruned))
	}

	// complicated node graph
	node = testSetup()
	nodes = node.Nodes()

	// set prune time to 100ms and wait for half of it
	pruneTime = 100 * time.Millisecond
	time.Sleep(pruneTime)

	// update the time of peer1
	node.peers["peer1"].lastSeen = time.Now()

	// should prune all but the root nodes and peer1
	pruned = node.PruneStalePeers(pruneTime)

	if len(pruned) != len(nodes)-2 {
		t.Errorf("Expected pruned node count: %d, got: %d", len(nodes)-2, len(pruned))
	}
}

func TestUnpackPeerTopology(t *testing.T) {
	pbPeer := &pb.Peer{
		Node: &pb.Node{
			Id:      "newPeer",
			Address: "newPeerAddress",
			Status: &pb.Status{
				Error: &pb.Error{},
			},
		},
		Peers: make([]*pb.Peer, 0),
	}
	// it should add pbPeer to the single node peers
	peer := UnpackPeerTopology(pbPeer, time.Now(), 5)
	if peer.id != pbPeer.Node.Id {
		t.Errorf("Expected peer id %s, found: %s", pbPeer.Node.Id, peer.id)
	}

	node := testSetup()
	// build a simple topology to update node peer1
	peer1 := node.peers["peer1"]
	pbPeer1Node := &pb.Node{
		Id:      peer1.id,
		Address: peer1.address,
		Status: &pb.Status{
			Error: &pb.Error{},
		},
	}

	pbPeer111 := &pb.Peer{
		Node: &pb.Node{
			Id:      "peer111",
			Address: "peer111Address",
			Status: &pb.Status{
				Error: &pb.Error{},
			},
		},
		Peers: make([]*pb.Peer, 0),
	}

	pbPeer121 := &pb.Peer{
		Node: &pb.Node{
			Id:      "peer121",
			Address: "peer121Address",
			Status: &pb.Status{
				Error: &pb.Error{},
			},
		},
		Peers: make([]*pb.Peer, 0),
	}
	// topology to update
	pbPeer1 := &pb.Peer{
		Node:  pbPeer1Node,
		Peers: []*pb.Peer{pbPeer111, pbPeer121},
	}
	// unpack peer1 topology
	peer = UnpackPeerTopology(pbPeer1, time.Now(), 5)
	// make sure peer1 topology has been correctly updated
	newPeerIds := []string{pbPeer111.Node.Id, pbPeer121.Node.Id}
	for _, id := range newPeerIds {
		if _, ok := peer.peers[id]; !ok {
			t.Errorf("Expected %s to be a peer of %s", id, "peer1")
		}
	}
}

func TestPeersToProto(t *testing.T) {
	// single node
	single := &node{
		id:      testNodeId,
		address: testNodeAddress,
		peers:   make(map[string]*node),
		network: newNetwork(Name(testNodeNetName)),
		status:  newStatus(),
	}
	topCount := 0

	protoPeers := PeersToProto(single, 0)

	if len(protoPeers.Peers) != topCount {
		t.Errorf("Expected to find %d nodes, found: %d", topCount, len(protoPeers.Peers))
	}

	// complicated node graph
	node := testSetup()
	topCount = 3
	// list of ids of nodes of depth 1 i.e. node peers
	peerIds := make(map[string]bool)
	// add peer Ids
	for _, id := range testNodePeerIds {
		peerIds[id] = true
	}
	// depth 1 should give us immmediate neighbours only
	protoPeers = PeersToProto(node, 1)

	if len(protoPeers.Peers) != topCount {
		t.Errorf("Expected to find %d nodes, found: %d", topCount, len(protoPeers.Peers))
	}
}
