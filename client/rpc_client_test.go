package client

import (
	"context"
	"fmt"
	"testing"

	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/mock"
	"github.com/micro/go-micro/selector"
)

func TestCallAddress(t *testing.T) {
	var called bool
	service := "test.service"
	method := "Test.Method"
	address := "10.1.10.1:8080"

	wrap := func(cf CallFunc) CallFunc {
		return func(ctx context.Context, addr string, req Request, rsp interface{}, opts CallOptions) error {
			called = true

			if req.Service() != service {
				return fmt.Errorf("expected service: %s got %s", service, req.Service())
			}

			if req.Method() != method {
				return fmt.Errorf("expected service: %s got %s", method, req.Method())
			}

			if addr != address {
				return fmt.Errorf("expected address: %s got %s", address, addr)
			}

			// don't do the call
			return nil
		}
	}

	r := mock.NewRegistry()
	c := NewClient(
		Registry(r),
		WrapCall(wrap),
	)
	c.Options().Selector.Init(selector.Registry(r))

	req := c.NewRequest(service, method, nil)

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
	method := "Test.Method"
	address := "10.1.10.1:8080"

	var called int

	wrap := func(cf CallFunc) CallFunc {
		return func(ctx context.Context, addr string, req Request, rsp interface{}, opts CallOptions) error {
			called++
			if called == 1 {
				return errors.InternalServerError("test.error", "retry request")
			}

			// don't do the call
			return nil
		}
	}

	r := mock.NewRegistry()
	c := NewClient(
		Registry(r),
		WrapCall(wrap),
	)
	c.Options().Selector.Init(selector.Registry(r))

	req := c.NewRequest(service, method, nil)

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
	method := "Test.Method"
	host := "10.1.10.1"
	port := 8080
	address := "10.1.10.1:8080"

	wrap := func(cf CallFunc) CallFunc {
		return func(ctx context.Context, addr string, req Request, rsp interface{}, opts CallOptions) error {
			called = true

			if req.Service() != service {
				return fmt.Errorf("expected service: %s got %s", service, req.Service())
			}

			if req.Method() != method {
				return fmt.Errorf("expected service: %s got %s", method, req.Method())
			}

			if addr != address {
				return fmt.Errorf("expected address: %s got %s", address, addr)
			}

			// don't do the call
			return nil
		}
	}

	r := mock.NewRegistry()
	c := NewClient(
		Registry(r),
		WrapCall(wrap),
	)
	c.Options().Selector.Init(selector.Registry(r))

	r.Register(&registry.Service{
		Name:    service,
		Version: "latest",
		Nodes: []*registry.Node{
			&registry.Node{
				Id:      id,
				Address: host,
				Port:    port,
			},
		},
	})

	req := c.NewRequest(service, method, nil)
	if err := c.Call(context.Background(), req, nil); err != nil {
		t.Fatal("call wrapper error", err)
	}

	if !called {
		t.Fatal("wrapper not called")
	}
}
