package client

import (
	"testing"

	"github.com/micro/go-micro/registry"
	"golang.org/x/net/context"
)

type mockRegistry struct{}

func (m *mockRegistry) GetService(service string) ([]*registry.Service, error) {
	return []*registry.Service{
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
	}, nil
}

func (m *mockRegistry) ListServices() ([]*registry.Service, error) {
	return []*registry.Service{}, nil
}

func (m *mockRegistry) Register(s *registry.Service) error {
	return nil
}

func (m *mockRegistry) Deregister(s *registry.Service) error {
	return nil
}

func (m *mockRegistry) Watch() (registry.Watcher, error) {
	return nil, nil
}

func TestNodeSelector(t *testing.T) {
	counts := map[string]int{}
	n := &nodeSelector{
		&mockRegistry{},
	}

	for i := 0; i < 100; i++ {
		n, err := n.Select(context.Background(), newRpcRequest("foo", "Foo.Bar", nil, ""))
		if err != nil {
			t.Errorf("Expected node, got err: %v", err)
		}
		counts[n.Id]++
	}

	t.Logf("Counts %v", counts)
}
