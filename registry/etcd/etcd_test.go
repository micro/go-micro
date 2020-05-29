package etcd

import (
	"os"
	"testing"

	"github.com/micro/go-micro/v2/registry"
)

func TestDomain(t *testing.T) {
	if travis := os.Getenv("TRAVIS"); travis == "true" {
		t.Skip()
	}

	testSrv := &registry.Service{
		Name:    "test1",
		Version: "1.0.1",
		Nodes: []*registry.Node{
			{
				Id:      "test1-1",
				Address: "10.0.0.1:10001",
				Metadata: map[string]string{
					"foo": "bar",
				},
			},
		},
	}

	r := NewRegistry(registry.Domain("foo"))
	if err := r.Register(testSrv); err != nil {
		t.Fatalf("Error registering service: %v", err)
	}

	t.Run("NoOption", func(t *testing.T) {
		srvs, err := r.GetService(testSrv.Name)
		if len(srvs) != 1 {
			t.Errorf("Expected 1 service, got %v", len(srvs))
		}
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	})

	t.Run("MatchingOption", func(t *testing.T) {
		srvs, err := r.GetService(testSrv.Name, registry.GetDomain("foo"))
		if len(srvs) != 1 {
			t.Errorf("Expected 1 service, got %v", len(srvs))
		}
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	})

	t.Run("NotMatchingOption", func(t *testing.T) {
		srvs, err := r.GetService(testSrv.Name, registry.GetDomain("bar"))
		if len(srvs) != 0 {
			t.Errorf("Expected 0 services, got %v", len(srvs))
		}
		if err != registry.ErrNotFound {
			t.Errorf("Expected %v, got %v", registry.ErrNotFound, err)
		}
	})
}
