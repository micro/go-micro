package client

import (
	"context"
	"fmt"
	"testing"

	"go-micro.dev/v5/errors"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/selector"
)

const (
	serviceName     = "test.service"
	serviceEndpoint = "Test.Endpoint"
)

func newTestRegistry() registry.Registry {
	return registry.NewMemoryRegistry(registry.Services(testData))
}

func TestCallAddress(t *testing.T) {
	var called bool
	service := serviceName
	endpoint := serviceEndpoint
	address := "10.1.10.1:8080"

	wrap := func(cf CallFunc) CallFunc {
		return func(_ context.Context, node *registry.Node, req Request, _ interface{}, _ CallOptions) error {
			called = true

			if req.Service() != service {
				return fmt.Errorf("expected service: %s got %s", service, req.Service())
			}

			if req.Endpoint() != endpoint {
				return fmt.Errorf("expected service: %s got %s", endpoint, req.Endpoint())
			}

			if node.Address != address {
				return fmt.Errorf("expected address: %s got %s", address, node.Address)
			}

			// don't do the call
			return nil
		}
	}

	r := newTestRegistry()
	c := NewClient(
		Registry(r),
		WrapCall(wrap),
	)

	if err := c.Options().Selector.Init(selector.Registry(r)); err != nil {
		t.Fatal("failed to initialize selector", err)
	}

	req := c.NewRequest(service, endpoint, nil)

	// test calling remote address
	if err := c.Call(context.Background(), req, nil, WithAddress(address)); err != nil {
		t.Fatal("call with address error", err)
	}

	if !called {
		t.Fatal("wrapper not called")
	}
}

func TestCallRetry(t *testing.T) {
	service := "test.service"
	endpoint := "Test.Endpoint"
	address := "10.1.10.1"

	var called int

	wrap := func(cf CallFunc) CallFunc {
		return func(_ context.Context, _ *registry.Node, _ Request, _ interface{}, _ CallOptions) error {
			called++
			if called == 1 {
				return errors.InternalServerError("test.error", "retry request")
			}
			// don't do the call
			return nil
		}
	}

	r := newTestRegistry()
	c := NewClient(
		Registry(r),
		WrapCall(wrap),
		Retry(RetryAlways),
		Retries(1),
	)

	if err := c.Options().Selector.Init(selector.Registry(r)); err != nil {
		t.Fatal("failed to initialize selector", err)
	}

	req := c.NewRequest(service, endpoint, nil)

	// test calling remote address
	if err := c.Call(context.Background(), req, nil, WithAddress(address)); err != nil {
		t.Fatal("call with address error", err)
	}

	// num calls
	if called < c.Options().CallOptions.Retries+1 {
		t.Fatal("request not retried")
	}
}

func TestCallWrapper(t *testing.T) {
	var called bool
	id := "test.1"
	service := "test.service"
	endpoint := "Test.Endpoint"
	address := "10.1.10.1:8080"

	wrap := func(cf CallFunc) CallFunc {
		return func(_ context.Context, node *registry.Node, req Request, _ interface{}, _ CallOptions) error {
			called = true

			if req.Service() != service {
				return fmt.Errorf("expected service: %s got %s", service, req.Service())
			}

			if req.Endpoint() != endpoint {
				return fmt.Errorf("expected service: %s got %s", endpoint, req.Endpoint())
			}

			if node.Address != address {
				return fmt.Errorf("expected address: %s got %s", address, node.Address)
			}

			// don't do the call
			return nil
		}
	}

	r := newTestRegistry()
	c := NewClient(
		Registry(r),
		WrapCall(wrap),
	)

	if err := c.Options().Selector.Init(selector.Registry(r)); err != nil {
		t.Fatal("failed to initialize selector", err)
	}

	err := r.Register(&registry.Service{
		Name:    service,
		Version: "latest",
		Nodes: []*registry.Node{
			{
				Id:      id,
				Address: address,
				Metadata: map[string]string{
					"protocol": "mucp",
				},
			},
		},
	})
	if err != nil {
		t.Fatal("failed to register service", err)
	}

	req := c.NewRequest(service, endpoint, nil)
	if err := c.Call(context.Background(), req, nil); err != nil {
		t.Fatal("call wrapper error", err)
	}

	if !called {
		t.Fatal("wrapper not called")
	}
}
