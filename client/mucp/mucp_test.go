package mucp

import (
	"context"
	"fmt"
	"testing"

	"github.com/micro/go-micro/v3/client"
	"github.com/micro/go-micro/v3/errors"
	"github.com/micro/go-micro/v3/registry"
	"github.com/micro/go-micro/v3/registry/memory"
)

func newTestRegistry() registry.Registry {
	return memory.NewRegistry(memory.Services(testData))
}

func TestCallAddress(t *testing.T) {
	var called bool
	service := "test.service"
	endpoint := "Test.Endpoint"
	address := "10.1.10.1:8080"

	wrap := func(cf client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
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
		client.Registry(r),
		client.WrapCall(wrap),
	)

	req := c.NewRequest(service, endpoint, nil)

	// test calling remote address
	if err := c.Call(context.Background(), req, nil, client.WithAddress(address)); err != nil {
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

	wrap := func(cf client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
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
		client.Registry(r),
		client.WrapCall(wrap),
	)

	req := c.NewRequest(service, endpoint, nil)

	// test calling remote address
	if err := c.Call(context.Background(), req, nil, client.WithAddress(address)); err != nil {
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

	wrap := func(cf client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
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
		client.Registry(r),
		client.WrapCall(wrap),
	)

	r.Register(&registry.Service{
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

	req := c.NewRequest(service, endpoint, nil)
	if err := c.Call(context.Background(), req, nil); err != nil {
		t.Fatal("call wrapper error", err)
	}

	if !called {
		t.Fatal("wrapper not called")
	}
}
