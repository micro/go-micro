package grpc

import (
	"context"
	"testing"
	"time"

	"go-micro.dev/v6/client"
	"go-micro.dev/v6/registry"
)

func TestEndpointBudgetNativeGRPC(t *testing.T) {
	var timeout time.Duration
	wrapper := func(client.CallFunc) client.CallFunc {
		return func(ctx context.Context, _ *registry.Node, _ client.Request, _ interface{}, opts client.CallOptions) error {
			timeout = opts.RequestTimeout
			return nil
		}
	}
	c := NewClient(client.EndpointOptions(map[string][]client.CallOption{"assistant/Agent.Chat": {client.WithRequestTimeout(time.Second)}}), client.WrapCall(wrapper))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := c.Call(ctx, c.NewRequest("assistant", "Agent.Chat", new(string)), new(string), client.WithAddress("unused:1")); err != nil {
		t.Fatal(err)
	}
	if timeout <= 0 || timeout > time.Second {
		t.Fatalf("endpoint budget overridden by caller: %v", timeout)
	}
}
