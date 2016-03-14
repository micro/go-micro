package mock

import (
	"testing"

	"github.com/micro/go-micro/registry"
)

var (
	testData = map[string][]*registry.Service{
		"foo": []*registry.Service{
			{
				Name:    "foo",
				Version: "1.0.0",
				Nodes: []*registry.Node{
					{
						Id:      "foo-1.0.0-123",
						Address: "localhost",
						Port:    9999,
					},
					{
						Id:      "foo-1.0.0-321",
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
						Id:      "foo-1.0.1-321",
						Address: "localhost",
						Port:    6666,
					},
				},
			},
			{
				Name:    "foo",
				Version: "1.0.3",
				Nodes: []*registry.Node{
					{
						Id:      "foo-1.0.3-345",
						Address: "localhost",
						Port:    8888,
					},
				},
			},
		},
		"bar": []*registry.Service{
			{
				Name:    "bar",
				Version: "default",
				Nodes: []*registry.Node{
					{
						Id:      "bar-1.0.0-123",
						Address: "localhost",
						Port:    9999,
					},
					{
						Id:      "bar-1.0.0-321",
						Address: "localhost",
						Port:    9999,
					},
				},
			},
			{
				Name:    "bar",
				Version: "latest",
				Nodes: []*registry.Node{
					{
						Id:      "bar-1.0.1-321",
						Address: "localhost",
						Port:    6666,
					},
				},
			},
		},
	}
)

func TestMockRegistry(t *testing.T) {
	m := NewRegistry()

	fn := func(k string, v []*registry.Service) {
		services, err := m.GetService(k)
		if err != nil {
			t.Errorf("Unexpected error getting service %s: %v", k, err)
		}

		if len(services) != len(v) {
			t.Errorf("Expected %d services for %s, got %d", len(v), k, len(services))
		}

		for _, service := range v {
			var seen bool
			for _, s := range services {
				if s.Version == service.Version {
					seen = true
					break
				}
			}
			if !seen {
				t.Errorf("expected to find version %s", service.Version)
			}
		}
	}

	// test existing mock data
	for k, v := range mockData {
		fn(k, v)
	}

	// register data
	for _, v := range testData {
		for _, service := range v {
			if err := m.Register(service); err != nil {
				t.Errorf("Unexpected register error: %v", err)
			}
		}
	}

	// using test data
	for k, v := range testData {

		fn(k, v)
	}

	// deregister
	for _, v := range testData {
		for _, service := range v {
			if err := m.Deregister(service); err != nil {
				t.Errorf("Unexpected deregister error: %v", err)
			}
		}
	}
}
