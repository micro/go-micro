package health

import (
	"context"
	"errors"
	"testing"

	"go-micro.dev/v5/registry"
)

// mockRegistry implements registry.Registry for testing.
type mockRegistry struct {
	listErr error
}

func (m *mockRegistry) Init(...registry.Option) error                                   { return nil }
func (m *mockRegistry) Options() registry.Options                                       { return registry.Options{} }
func (m *mockRegistry) Register(*registry.Service, ...registry.RegisterOption) error    { return nil }
func (m *mockRegistry) Deregister(*registry.Service, ...registry.DeregisterOption) error { return nil }
func (m *mockRegistry) GetService(string, ...registry.GetOption) ([]*registry.Service, error) {
	return nil, nil
}
func (m *mockRegistry) ListServices(...registry.ListOption) ([]*registry.Service, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []*registry.Service{{Name: "test"}}, nil
}
func (m *mockRegistry) Watch(...registry.WatchOption) (registry.Watcher, error) { return nil, nil }
func (m *mockRegistry) String() string                                          { return "mock" }

func TestRegistryCheck_Healthy(t *testing.T) {
	check := RegistryCheck(&mockRegistry{})
	err := check(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestRegistryCheck_Unhealthy(t *testing.T) {
	check := RegistryCheck(&mockRegistry{listErr: errors.New("connection refused")})
	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestRegistryCheck_NilRegistry(t *testing.T) {
	check := RegistryCheck(nil)
	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for nil registry, got nil")
	}
}

func TestRegistryCheck_Integration(t *testing.T) {
	Reset()

	reg := &mockRegistry{}
	Register("registry", RegistryCheck(reg))

	resp := Run(context.Background())
	if resp.Status != StatusUp {
		t.Errorf("expected status up, got %s", resp.Status)
	}

	// Now simulate disconnection
	Reset()
	reg.listErr = errors.New("etcd connection lost")
	Register("registry", RegistryCheck(reg))

	resp = Run(context.Background())
	if resp.Status != StatusDown {
		t.Errorf("expected status down, got %s", resp.Status)
	}
	if resp.Checks[0].Error == "" {
		t.Error("expected error message in check result")
	}
}
