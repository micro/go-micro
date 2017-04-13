package client

import (
	"fmt"
	"testing"
	"time"

	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/mock"
	"github.com/micro/go-micro/selector"

	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
)

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

func TestCall_MetadataRaceCondition(t *testing.T) {
	r := mock.NewRegistry()
	c := NewClient(Registry(r), DialTimeout(10*time.Millisecond))
	c.Options().Selector.Init(selector.Registry(r))

	r.Register(&registry.Service{
		Name:    "micro.service",
		Version: "latest",
		Nodes: []*registry.Node{
			&registry.Node{
				Id:      "id1",
				Address: "10.1.10.1",
				Port:    1234,
			},
		},
	})

	ctx := metadata.NewContext(context.Background(), metadata.Metadata{})
	req := c.NewRequest("micro.service", "TestService.Method", nil)

	g, _ := errgroup.WithContext(ctx)

	g.Go(func() error {
		return c.Call(ctx, req, nil)
	})

	g.Go(func() error {
		md, ok := metadata.FromContext(ctx)
		if !ok {
			t.Fatal("unable to parse metadata from context")
		}
		md["key"] = "value"

		return c.Call(metadata.NewContext(ctx, md), req, nil)
	})

	g.Wait()
}
