package memory

import (
	"fmt"
	"testing"
	"time"

	"github.com/micro/go-micro/registry"
)

var (
	testData = map[string][]*registry.Service{
		"foo": {
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
		"bar": {
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

func TestMemoryRegistryTTL(t *testing.T) {
	m := NewContextRegistry()

	for _, v := range testData {
		for _, service := range v {
			if err := m.Register(service, registry.RegisterTTL(time.Millisecond)); err != nil {
				t.Fatal(err)
			}
		}
	}

	time.Sleep(time.Millisecond)

	for name := range testData {
		svcs, err := m.GetService(name)
		if err != nil {
			t.Fatal(err)
		}

		for _, svc := range svcs {
			if len(svc.Nodes) > 0 {
				t.Fatalf("Service %q still has nodes registered", name)
			}
		}
	}
}

func TestMemoryRegistryTTLConcurrent(t *testing.T) {
	concurrency := 1000
	waitTime := time.Second
	m := NewContextRegistry()

	for _, v := range testData {
		for _, service := range v {
			if err := m.Register(service, registry.RegisterTTL(waitTime)); err != nil {
				t.Fatal(err)
			}
		}
	}

	t.Logf("test will wait %v, then check TTL timeouts", waitTime)

	errChan := make(chan error, concurrency)
	syncChan := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		go func() {
			<-syncChan
			for name := range testData {
				svcs, err := m.GetService(name)
				if err != nil {
					errChan <- err
					return
				}

				for _, svc := range svcs {
					if len(svc.Nodes) > 0 {
						errChan <- fmt.Errorf("Service %q still has nodes registered", name)
						return
					}
				}
			}

			errChan <- nil
		}()
	}

	time.Sleep(waitTime)
	close(syncChan)

	for i := 0; i < concurrency; i++ {
		if err := <-errChan; err != nil {
			t.Fatal(err)
		}
	}
}
