package network

import (
	"testing"
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

func TestPeers(t *testing.T) {
	// single node
	single := &node{
		id:      testNodeId,
		address: testNodeAddress,
		peers:   make(map[string]*node),
		network: newNetwork(Name(testNodeNetName)),
	}
	// get all the nodes including yourself
	peers := single.Peers()
	peerCount := 0

	if len(peers) != peerCount {
		t.Errorf("Expected to find %d peers, found: %d", peerCount, len(peers))
	}

	// complicated node graph
	node := testSetup()
	// get all the nodes including yourself
	peers = node.Peers()

	// compile a list of ids of all nodes in the network into map for easy indexing
	peerIds := make(map[string]bool)
	// add peer Ids
	for _, id := range testNodePeerIds {
		peerIds[id] = true
	}

	// we should return the correct number of peers
	if len(peers) != len(peerIds) {
		t.Errorf("Expected %d nodes, found: %d", len(peerIds), len(peers))
	}

	// iterate through the list of peers and makes sure all have been returned
	for _, peer := range peers {
		if _, ok := peerIds[peer.Id()]; !ok {
			t.Errorf("Expected to find %s peer", peer.Id())
		}
	}
}

func TestTopology(t *testing.T) {
	// single node
	single := &node{
		id:      testNodeId,
		address: testNodeAddress,
		peers:   make(map[string]*node),
		network: newNetwork(Name(testNodeNetName)),
	}
	// get all the nodes including yourself
	topology := single.Topology(MaxDepth)
	// you should not be in your topology
	topCount := 0

	if len(topology.peers) != topCount {
		t.Errorf("Expected to find %d nodes, found: %d", topCount, len(topology.peers))
	}

	// complicated node graph
	node := testSetup()
	// list of ids of nodes of depth 1 i.e. node peers
	peerIds := make(map[string]bool)
	// add peer Ids
	for _, id := range testNodePeerIds {
		peerIds[id] = true
	}
	topology = node.Topology(1)

	// depth 1 should return only immediate peers
	if len(topology.peers) != len(peerIds) {
		t.Errorf("Expected to find %d nodes, found: %d", len(peerIds), len(topology.peers))
	}
	for id := range topology.peers {
		if _, ok := peerIds[id]; !ok {
			t.Errorf("Expected to find %s peer", id)
		}
	}

	// add peers of peers to peerIds
	for _, id := range testPeerOfPeerIds {
		peerIds[id] = true
	}
	topology = node.Topology(2)

	// iterate through the whole graph
	// NOTE: this is a manual iteration as we know the size of the graph
	for id, peer := range topology.peers {
		if _, ok := peerIds[id]; !ok {
			t.Errorf("Expected to find %s peer", peer.Id())
		}
		// peers of peers
		for id := range peer.peers {
			if _, ok := peerIds[id]; !ok {
				t.Errorf("Expected to find %s peer", peer.Id())
			}
		}
	}
}
