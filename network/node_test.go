package network

import (
	"testing"

	pb "github.com/micro/go-micro/network/proto"
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
	}

	// add some peers to the node
	for _, id := range testNodePeerIds {
		testNode.peers[id] = &node{
			id:      id,
			address: testNode.address + "-" + id,
			peers:   make(map[string]*node),
			network: testNode.network,
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

func TestUpdatePeerTopology(t *testing.T) {
	// single node
	single := &node{
		id:      testNodeId,
		address: testNodeAddress,
		peers:   make(map[string]*node),
		network: newNetwork(Name(testNodeNetName)),
	}
	// nil peer should return error
	if err := single.updatePeerTopology(nil, 5); err == nil {
		t.Errorf("Expected error, got %s", err)
	}

	// update with peer that is not yet in the peer map
	pbPeer := &pb.Peer{
		Node: &pb.Node{
			Id:      "newPeer",
			Address: "newPeerAddress",
		},
		Peers: make([]*pb.Peer, 0),
	}
	// it should add pbPeer to the single node peers
	if err := single.updatePeerTopology(pbPeer, 5); err != nil {
		t.Errorf("Error updating topology: %s", err)
	}
	if _, ok := single.peers[pbPeer.Node.Id]; !ok {
		t.Errorf("Expected %s to be added to %s peers", pbPeer.Node.Id, single.id)
	}

	// complicated node graph
	node := testSetup()
	// build a simple topology to update node peer1
	peer1 := node.peers["peer1"]
	pbPeer1Node := &pb.Node{
		Id:      peer1.id,
		Address: peer1.address,
	}

	pbPeer111 := &pb.Peer{
		Node: &pb.Node{
			Id:      "peer111",
			Address: "peer111Address",
		},
		Peers: make([]*pb.Peer, 0),
	}

	pbPeer121 := &pb.Peer{
		Node: &pb.Node{
			Id:      "peer121",
			Address: "peer121Address",
		},
		Peers: make([]*pb.Peer, 0),
	}
	// topology to update
	pbPeer1 := &pb.Peer{
		Node:  pbPeer1Node,
		Peers: []*pb.Peer{pbPeer111, pbPeer121},
	}
	// update peer1 topology
	if err := node.updatePeerTopology(pbPeer1, 5); err != nil {
		t.Errorf("Error updating topology: %s", err)
	}
	// make sure peer1 topology has been correctly updated
	newPeerIds := []string{pbPeer111.Node.Id, pbPeer121.Node.Id}
	for _, id := range newPeerIds {
		if _, ok := node.peers["peer1"].peers[id]; !ok {
			t.Errorf("Expected %s to be a peer of %s", id, "peer1")
		}
	}
}

func TestGetProtoTopology(t *testing.T) {
	// single node
	single := &node{
		id:      testNodeId,
		address: testNodeAddress,
		peers:   make(map[string]*node),
		network: newNetwork(Name(testNodeNetName)),
	}
	topCount := 0

	protoTop, err := single.getProtoTopology(10)
	if err != nil {
		t.Errorf("Error getting proto topology: %s", err)
	}
	if len(protoTop.Peers) != topCount {
		t.Errorf("Expected to find %d nodes, found: %d", topCount, len(protoTop.Peers))
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
	protoTop, err = node.getProtoTopology(1)
	if err != nil {
		t.Errorf("Error getting proto topology: %s", err)
	}
	if len(protoTop.Peers) != topCount {
		t.Errorf("Expected to find %d nodes, found: %d", topCount, len(protoTop.Peers))
	}
}
