package client

import (
	"testing"

	"github.com/piemapping/go-micro/registry"
)

func TestNodeSelector(t *testing.T) {
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
			Version: "1.0.1",
			Nodes: []*registry.Node{
				{
					Id:      "foo-321",
					Address: "localhost",
					Port:    6666,
				},
			},
		},
	}

	counts := map[string]int{}

	for i := 0; i < 100; i++ {
		n, err := nodeSelector(services)
		if err != nil {
			t.Errorf("Expected node, got err: %v", err)
		}
		counts[n.Id]++
	}

	t.Logf("Counts %v", counts)
}
