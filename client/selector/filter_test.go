package selector

import (
	"testing"

	"github.com/micro/go-micro/v2/registry"
)

func TestFilterEndpoint(t *testing.T) {
	testData := []struct {
		services []*registry.Service
		endpoint string
		count    int
	}{
		{
			services: []*registry.Service{
				{
					Name:    "test",
					Version: "1.0.0",
					Endpoints: []*registry.Endpoint{
						{
							Name: "Foo.Bar",
						},
					},
				},
				{
					Name:    "test",
					Version: "1.1.0",
					Endpoints: []*registry.Endpoint{
						{
							Name: "Baz.Bar",
						},
					},
				},
			},
			endpoint: "Foo.Bar",
			count:    1,
		},
		{
			services: []*registry.Service{
				{
					Name:    "test",
					Version: "1.0.0",
					Endpoints: []*registry.Endpoint{
						{
							Name: "Foo.Bar",
						},
					},
				},
				{
					Name:    "test",
					Version: "1.1.0",
					Endpoints: []*registry.Endpoint{
						{
							Name: "Foo.Bar",
						},
					},
				},
			},
			endpoint: "Bar.Baz",
			count:    0,
		},
	}

	for _, data := range testData {
		filter := FilterEndpoint(data.endpoint)
		services := filter(data.services)

		if len(services) != data.count {
			t.Fatalf("Expected %d services, got %d", data.count, len(services))
		}

		for _, service := range services {
			var seen bool

			for _, ep := range service.Endpoints {
				if ep.Name == data.endpoint {
					seen = true
					break
				}
			}

			if !seen && data.count > 0 {
				t.Fatalf("Expected %d services but seen is %t; result %+v", data.count, seen, services)
			}
		}
	}
}

func TestFilterLabel(t *testing.T) {
	testData := []struct {
		services []*registry.Service
		label    [2]string
		count    int
	}{
		{
			services: []*registry.Service{
				{
					Name:    "test",
					Version: "1.0.0",
					Nodes: []*registry.Node{
						{
							Id:      "test-1",
							Address: "localhost",
							Metadata: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
				{
					Name:    "test",
					Version: "1.1.0",
					Nodes: []*registry.Node{
						{
							Id:      "test-2",
							Address: "localhost",
							Metadata: map[string]string{
								"foo": "baz",
							},
						},
					},
				},
			},
			label: [2]string{"foo", "bar"},
			count: 1,
		},
		{
			services: []*registry.Service{
				{
					Name:    "test",
					Version: "1.0.0",
					Nodes: []*registry.Node{
						{
							Id:      "test-1",
							Address: "localhost",
						},
					},
				},
				{
					Name:    "test",
					Version: "1.1.0",
					Nodes: []*registry.Node{
						{
							Id:      "test-2",
							Address: "localhost",
						},
					},
				},
			},
			label: [2]string{"foo", "bar"},
			count: 0,
		},
	}

	for _, data := range testData {
		filter := FilterLabel(data.label[0], data.label[1])
		services := filter(data.services)

		if len(services) != data.count {
			t.Fatalf("Expected %d services, got %d", data.count, len(services))
		}

		for _, service := range services {
			var seen bool

			for _, node := range service.Nodes {
				if node.Metadata[data.label[0]] != data.label[1] {
					t.Fatalf("Expected %s=%s but got %s=%s for service %+v node %+v",
						data.label[0], data.label[1], data.label[0], node.Metadata[data.label[0]], service, node)
				}
				seen = true
			}

			if !seen {
				t.Fatalf("Expected node for %s=%s but saw none; results %+v", data.label[0], data.label[1], service)
			}
		}
	}
}

func TestFilterVersion(t *testing.T) {
	testData := []struct {
		services []*registry.Service
		version  string
		count    int
	}{
		{
			services: []*registry.Service{
				{
					Name:    "test",
					Version: "1.0.0",
				},
				{
					Name:    "test",
					Version: "1.1.0",
				},
			},
			version: "1.0.0",
			count:   1,
		},
		{
			services: []*registry.Service{
				{
					Name:    "test",
					Version: "1.0.0",
				},
				{
					Name:    "test",
					Version: "1.1.0",
				},
			},
			version: "2.0.0",
			count:   0,
		},
	}

	for _, data := range testData {
		filter := FilterVersion(data.version)
		services := filter(data.services)

		if len(services) != data.count {
			t.Fatalf("Expected %d services, got %d", data.count, len(services))
		}

		var seen bool

		for _, service := range services {
			if service.Version != data.version {
				t.Fatalf("Expected version %s, got %s", data.version, service.Version)
			}
			seen = true
		}

		if !seen && data.count > 0 {
			t.Fatalf("Expected %d services but seen is %t; result %+v", data.count, seen, services)
		}
	}
}
