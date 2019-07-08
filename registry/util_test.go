package registry

import (
	"testing"
)

func TestRemove(t *testing.T) {
	services := []*Service{
		{
			Name:    "foo",
			Version: "1.0.0",
			Nodes: []*Node{
				{
					Id:      "foo-123",
					Address: "localhost:9999",
				},
			},
		},
		{
			Name:    "foo",
			Version: "1.0.0",
			Nodes: []*Node{
				{
					Id:      "foo-123",
					Address: "localhost:6666",
				},
			},
		},
	}

	servs := Remove([]*Service{services[0]}, []*Service{services[1]})
	if i := len(servs); i > 0 {
		t.Errorf("Expected 0 nodes, got %d: %+v", i, servs)
	}
	t.Logf("Services %+v", servs)
}

func TestRemoveNodes(t *testing.T) {
	services := []*Service{
		{
			Name:    "foo",
			Version: "1.0.0",
			Nodes: []*Node{
				{
					Id:      "foo-123",
					Address: "localhost:9999",
				},
				{
					Id:      "foo-321",
					Address: "localhost:6666",
				},
			},
		},
		{
			Name:    "foo",
			Version: "1.0.0",
			Nodes: []*Node{
				{
					Id:      "foo-123",
					Address: "localhost:6666",
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
