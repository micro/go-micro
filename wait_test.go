package micro

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go-micro.dev/v6/registry"
)

type waitingRegistry struct {
	registry.Registry
	get func(string, ...registry.GetOption) ([]*registry.Service, error)
}

func (r waitingRegistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	return r.get(name, opts...)
}

func TestWaitForServiceRetriesDiscoveryAndProbe(t *testing.T) {
	var lookups, probes int
	transient := errors.New("not ready")
	reg := waitingRegistry{get: func(_ string, opts ...registry.GetOption) ([]*registry.Service, error) {
		lookups++
		var options registry.GetOptions
		for _, opt := range opts {
			opt(&options)
		}
		if options.Context == nil {
			t.Error("missing lookup context")
		}
		if lookups == 1 {
			return nil, transient
		}
		if lookups == 2 {
			return []*registry.Service{{Nodes: []*registry.Node{{Address: ""}}}}, nil
		}
		return []*registry.Service{{Nodes: []*registry.Node{{Address: "localhost:1"}}}}, nil
	}}
	svc := NewService("caller", Registry(reg))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := WaitForService(ctx, svc, "dependency", WaitBackoff(time.Millisecond, 2*time.Millisecond), WaitProbe(func(context.Context) error {
		probes++
		if probes == 1 {
			return transient
		}
		return nil
	}))
	if err != nil || lookups != 4 || probes != 2 {
		t.Fatalf("err=%v lookups=%d probes=%d", err, lookups, probes)
	}
}

func TestWaitForServiceCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	entered, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	var calls atomic.Int32
	reg := waitingRegistry{get: func(string, ...registry.GetOption) ([]*registry.Service, error) {
		calls.Add(1)
		close(entered)
		<-release
		return nil, registry.ErrNotFound
	}}
	svc := NewService("caller", Registry(reg))
	done := make(chan error, 1)
	go func() { done <- WaitForService(ctx, svc, "dependency") }()
	<-entered
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation blocked behind registry")
	}
	if calls.Load() != 1 {
		t.Fatal("unexpected extra lookup")
	}
}

func TestWaitForServiceCanceledBeforeLookup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reg := waitingRegistry{get: func(string, ...registry.GetOption) ([]*registry.Service, error) {
		t.Error("lookup after cancellation")
		return nil, nil
	}}
	if err := WaitForService(ctx, NewService("caller", Registry(reg)), "dependency"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestWaitForServiceDeadlinePreservesDiscoveryError(t *testing.T) {
	cause := errors.New("registry unavailable")
	reg := waitingRegistry{get: func(string, ...registry.GetOption) ([]*registry.Service, error) { return nil, cause }}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := WaitForService(ctx, NewService("caller", Registry(reg)), "dependency", WaitBackoff(time.Millisecond, time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, cause) {
		t.Fatalf("lost error cause: %v", err)
	}
}
