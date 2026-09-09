package health

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/registry"
)

// fakeRegistry implements registry.Registry with a configurable
// ListServices result, for exercising RegistryCheck.
type fakeRegistry struct {
	listErr error
	block   chan struct{} // if non-nil, ListServices blocks until it is closed
}

func (f *fakeRegistry) Init(...registry.Option) error { return nil }
func (f *fakeRegistry) Options() registry.Options     { return registry.Options{} }
func (f *fakeRegistry) Register(*registry.Service, ...registry.RegisterOption) error {
	return nil
}
func (f *fakeRegistry) Deregister(*registry.Service, ...registry.DeregisterOption) error {
	return nil
}
func (f *fakeRegistry) GetService(string, ...registry.GetOption) ([]*registry.Service, error) {
	return nil, nil
}
func (f *fakeRegistry) ListServices(...registry.ListOption) ([]*registry.Service, error) {
	if f.block != nil {
		<-f.block
	}
	return nil, f.listErr
}
func (f *fakeRegistry) Watch(...registry.WatchOption) (registry.Watcher, error) { return nil, nil }
func (f *fakeRegistry) String() string                                          { return "fake" }

func TestRegistryCheckNil(t *testing.T) {
	if err := RegistryCheck(nil)(context.Background()); err == nil {
		t.Fatal("a nil registry should fail the check")
	}
}

func TestRegistryCheckHealthy(t *testing.T) {
	check := RegistryCheck(registry.NewMemoryRegistry())
	if err := check(context.Background()); err != nil {
		t.Fatalf("a reachable registry should pass: %v", err)
	}
}

func TestRegistryCheckDown(t *testing.T) {
	check := RegistryCheck(&fakeRegistry{listErr: errors.New("connection refused")})
	err := check(context.Background())
	if err == nil {
		t.Fatal("the check should fail when the registry is unreachable")
	}
	if !strings.Contains(err.Error(), "unreachable") {
		t.Errorf("error should describe the registry as unreachable: %v", err)
	}
}

func TestRegistryCheckTimeout(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	check := RegistryCheck(&fakeRegistry{block: block})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := check(ctx)
	if err == nil {
		t.Fatal("the check should time out when the registry call hangs")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error should describe a timeout: %v", err)
	}
}

// RegistryCheck should integrate with the readiness machinery: a down
// registry registered as a (critical) check makes the service not ready.
func TestRegistryCheckMarksNotReady(t *testing.T) {
	Reset()
	defer Reset()

	Register("registry", RegistryCheck(&fakeRegistry{listErr: errors.New("etcd gone")}))
	if IsReady(context.Background()) {
		t.Error("service should be not-ready when the registry check is down")
	}
}

func TestRegistryServiceCheckHealthy(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	service := &registry.Service{
		Name:  "orders",
		Nodes: []*registry.Node{{Id: "orders-1"}},
	}
	if err := reg.Register(service); err != nil {
		t.Fatalf("register service: %v", err)
	}

	check := RegistryServiceCheck(reg, "orders", "orders-1")
	if err := check(context.Background()); err != nil {
		t.Fatalf("registered service node should pass: %v", err)
	}
}

func TestRegistryServiceCheckMissingNode(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	service := &registry.Service{
		Name:  "orders",
		Nodes: []*registry.Node{{Id: "orders-1"}},
	}
	if err := reg.Register(service); err != nil {
		t.Fatalf("register service: %v", err)
	}

	check := RegistryServiceCheck(reg, "orders", "orders-2")
	err := check(context.Background())
	if err == nil {
		t.Fatal("missing service node should fail")
	}
	if !strings.Contains(err.Error(), "missing node orders-2") {
		t.Errorf("error should describe the missing node: %v", err)
	}
}

func TestRegistryServiceCheckMissingService(t *testing.T) {
	check := RegistryServiceCheck(registry.NewMemoryRegistry(), "orders", "orders-1")
	err := check(context.Background())
	if err == nil {
		t.Fatal("missing service should fail")
	}
	if !strings.Contains(err.Error(), registry.ErrNotFound.Error()) {
		t.Errorf("error should include registry lookup failure: %v", err)
	}
}

func TestRegistryServiceCheckMarksNotReady(t *testing.T) {
	Reset()
	defer Reset()

	Register("registry-service", RegistryServiceCheck(registry.NewMemoryRegistry(), "orders", "orders-1"))
	if IsReady(context.Background()) {
		t.Error("service should be not-ready when its registry node is missing")
	}
}
