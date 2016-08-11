package registry

import (
	"testing"

	"github.com/hashicorp/consul/api"
)

func TestHealthyServiceHandler(t *testing.T) {
	watcher := newWatcher()
	serviceEntry := newServiceEntry(
		"node-name", "node-address", "service-name", "v1.0.0",
		[]*api.HealthCheck{
			newHealthCheck("node-name", "service-name", "passing"),
		},
	)

	watcher.serviceHandler(1234, []*api.ServiceEntry{serviceEntry})

	if len(watcher.services["service-name"][0].Nodes) != 1 {
		t.Errorf("Expected length of the service nodes to be 1")
	}
}

func TestUnhealthyServiceHandler(t *testing.T) {
	watcher := newWatcher()
	serviceEntry := newServiceEntry(
		"node-name", "node-address", "service-name", "v1.0.0",
		[]*api.HealthCheck{
			newHealthCheck("node-name", "service-name", "critical"),
		},
	)

	watcher.serviceHandler(1234, []*api.ServiceEntry{serviceEntry})

	if len(watcher.services["service-name"][0].Nodes) != 0 {
		t.Errorf("Expected length of the service nodes to be 0")
	}
}

func TestUnhealthyNodeServiceHandler(t *testing.T) {
	watcher := newWatcher()
	serviceEntry := newServiceEntry(
		"node-name", "node-address", "service-name", "v1.0.0",
		[]*api.HealthCheck{
			newHealthCheck("node-name", "service-name", "passing"),
			newHealthCheck("node-name", "serfHealth", "critical"),
		},
	)

	watcher.serviceHandler(1234, []*api.ServiceEntry{serviceEntry})

	if len(watcher.services["service-name"][0].Nodes) != 0 {
		t.Errorf("Expected length of the service nodes to be 0")
	}
}

func newWatcher() *consulWatcher {
	return &consulWatcher{
		exit:     make(chan bool),
		next:     make(chan *Result, 10),
		services: make(map[string][]*Service),
	}
}

func newHealthCheck(node, name, status string) *api.HealthCheck {
	return &api.HealthCheck{
		Node:        node,
		Name:        name,
		Status:      status,
		ServiceName: name,
	}
}

func newServiceEntry(node, address, name, version string, checks []*api.HealthCheck) *api.ServiceEntry {
	return &api.ServiceEntry{
		Node: &api.Node{Node: node, Address: name},
		Service: &api.AgentService{
			Service: name,
			Address: address,
			Tags:    encodeVersion(version),
		},
		Checks: checks,
	}
}
