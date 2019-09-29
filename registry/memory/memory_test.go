package memory

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
						Address: "localhost:9999",
					},
					{
						Id:      "foo-1.0.0-321",
						Address: "localhost:9999",
					},
				},
			},
			{
				Name:    "foo",
				Version: "1.0.1",
				Nodes: []*registry.Node{
					{
						Id:      "foo-1.0.1-321",
						Address: "localhost:6666",
					},
				},
			},
			{
				Name:    "foo",
				Version: "1.0.3",
				Nodes: []*registry.Node{
					{
						Id:      "foo-1.0.3-345",
						Address: "localhost:8888",
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
						Address: "localhost:9999",
					},
					{
						Id:      "bar-1.0.0-321",
						Address: "localhost:9999",
					},
				},
			},
			{
				Name:    "bar",
				Version: "latest",
				Nodes: []*registry.Node{
					{
						Id:      "bar-1.0.1-321",
						Address: "localhost:6666",
					},
				},
			},
		},
	}
)

func TestMemoryRegistry(t *testing.T) {
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

	// register data
	for _, v := range testData {
		serviceCount := 0
		for _, service := range v {
			if err := m.Register(service); err != nil {
				t.Errorf("Unexpected register error: %v", err)
			}
			serviceCount++
			// after the service has been registered we should be able to query it
			services, err := m.GetService(service.Name)
			if err != nil {
				t.Errorf("Unexpected error getting service %s: %v", service.Name, err)
			}
			if len(services) != serviceCount {
				t.Errorf("Expected %d services for %s, got %d", serviceCount, service.Name, len(services))
			}
		}
	}

	// using test data
	for k, v := range testData {
		fn(k, v)
	}

	services, err := m.ListServices()
	if err != nil {
		t.Errorf("Unexpected error when listing services: %v", err)
	}

	totalServiceCount := 0
	for _, testSvc := range testData {
		for range testSvc {
			totalServiceCount++
		}
	}

	if len(services) != totalServiceCount {
		t.Errorf("Expected total service count: %d, got: %d", totalServiceCount, len(services))
	}

	// deregister
	for _, v := range testData {
		for _, service := range v {
			if err := m.Deregister(service); err != nil {
				t.Errorf("Unexpected deregister error: %v", err)
			}
		}
	}

	// after all the service nodes have been deregistered we should not get any results
	for _, v := range testData {
		for _, service := range v {
			services, err := m.GetService(service.Name)
			if err != registry.ErrNotFound {
				t.Errorf("Expected error: %v, got: %v", registry.ErrNotFound, err)
			}
			if len(services) != 0 {
				t.Errorf("Expected %d services for %s, got %d", 0, service.Name, len(services))
			}
		}
	}
}
