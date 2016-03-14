package mock

import (
	"testing"

	"github.com/micro/go-micro/registry"
)

func TestDelServices(t *testing.T) {
	services := []*registry.Service{
		{
			Name:    "foo",
			Version: "1.0.0",
			Nodes: []*registry.Node{
				{
					Id:      "foo-123",
					Address: "localhost",
					Port:    9999,
				},
			},
		},
		{
			Name:    "foo",
			Version: "1.0.0",
			Nodes: []*registry.Node{
				{
					Id:      "foo-123",
					Address: "localhost",
					Port:    6666,
				},
			},
		},
	}

	servs := delServices([]*registry.Service{services[0]}, []*registry.Service{services[1]})
	if i := len(servs); i > 0 {
		t.Errorf("Expected 0 nodes, got %d: %+v", i, servs)
	}
	t.Logf("Services %+v", servs)
}

func TestDelNodes(t *testing.T) {
	services := []*registry.Service{
		{
			Name:    "foo",
			Version: "1.0.0",
			Nodes: []*registry.Node{
				{
					Id:      "foo-123",
					Address: "localhost",
					Port:    9999,
				},
				{
					Id:      "foo-321",
					Address: "localhost",
					Port:    6666,
				},
			},
		},
		{
			Name:    "foo",
			Version: "1.0.0",
			Nodes: []*registry.Node{
				{
					Id:      "foo-123",
					Address: "localhost",
					Port:    6666,
				},
			},
		},
	}

	nodes := delNodes(services[0].Nodes, services[1].Nodes)
	if i := len(nodes); i != 1 {
		t.Errorf("Expected only 1 node, got %d: %+v", i, nodes)
	}
	t.Logf("Nodes %+v", nodes)
}
