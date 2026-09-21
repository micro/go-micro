package client

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	merrors "go-micro.dev/v6/errors"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
)

func TestEndpointOptionsPrecedence(t *testing.T) {
	options := map[string][]CallOption{"assistant/Agent.Chat": {WithRequestTimeout(2 * time.Minute), WithConnectionTimeout(time.Minute), WithRetries(0)}}
	client := NewClient(RequestTimeout(time.Second), EndpointOptions(options))
	delete(options, "assistant/Agent.Chat")
	req := client.NewRequest("assistant", "Agent.Chat", nil)
	got := ResolveCallOptions(client.Options(), req)
	if got.RequestTimeout != 2*time.Minute || got.ConnectionTimeout != time.Minute || got.Retries != 0 {
		t.Fatalf("options=%+v", got)
	}
	got = ResolveCallOptions(client.Options(), req, WithRequestTimeout(3*time.Minute))
	if got.RequestTimeout != 3*time.Minute {
		t.Fatal("per-call option did not win")
	}
	got = ResolveCallOptions(client.Options(), client.NewRequest("other", "Agent.Chat", nil))
	if got.RequestTimeout != time.Second {
		t.Fatal("endpoint config leaked to other service")
	}
}

type discoverySelector struct {
	selector.Selector
	err error
}

func (s discoverySelector) Select(string, ...selector.SelectOption) (selector.Next, error) {
	return nil, s.err
}

func TestDiscoveryErrorStatusAndCause(t *testing.T) {
	for _, tc := range []struct {
		cause  error
		status int32
	}{{registry.ErrNotFound, 404}, {selector.ErrNotFound, 404}, {ErrNoNodes, 503}, {fmt.Errorf("backend: %w", registry.ErrUnavailable), 503}} {
		c := NewClient(Selector(discoverySelector{err: tc.cause}))
		err := c.Call(context.Background(), c.NewRequest("missing", "Service.Get", nil), new(string))
		var typed *DiscoveryError
		var status *merrors.Error
		if !errors.As(err, &typed) || !errors.As(err, &status) || status.Code != tc.status || !errors.Is(err, tc.cause) {
			t.Fatalf("cause=%v err=%v", tc.cause, err)
		}
	}
}

func TestEndpointRequestBudgetWithCallerDeadline(t *testing.T) {
	var timeout time.Duration
	wrapper := func(CallFunc) CallFunc {
		return func(ctx context.Context, _ *registry.Node, _ Request, _ interface{}, opts CallOptions) error {
			timeout = opts.RequestTimeout
			return nil
		}
	}
	c := NewClient(EndpointOptions(map[string][]CallOption{"assistant/Agent.Chat": {WithRequestTimeout(time.Second)}}), WrapCall(wrapper))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := c.Call(ctx, c.NewRequest("assistant", "Agent.Chat", nil), new(string), WithAddress("unused:1")); err != nil {
		t.Fatal(err)
	}
	if timeout <= 0 || timeout > time.Second {
		t.Fatalf("endpoint budget overridden by caller: %v", timeout)
	}
}
