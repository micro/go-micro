package service

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	svc := New(Name("test"))
	if svc == nil {
		t.Fatal("New returned nil")
	}
	if svc.Name() != "test" {
		t.Errorf("Name() = %q, want %q", svc.Name(), "test")
	}
	if svc.String() != "micro" {
		t.Errorf("String() = %q, want %q", svc.String(), "micro")
	}
}

func TestServiceComponents(t *testing.T) {
	svc := New(Name("components"), Address(":0"))
	if svc.Client() == nil {
		t.Error("Client() is nil")
	}
	if svc.Server() == nil {
		t.Error("Server() is nil")
	}
	if svc.Model() == nil {
		t.Error("Model() is nil")
	}
}

func TestServiceOptions(t *testing.T) {
	svc := New(Name("opts"), Address(":0"))
	opts := svc.Options()
	if opts.Server == nil {
		t.Error("Options().Server is nil")
	}
	if opts.Client == nil {
		t.Error("Options().Client is nil")
	}
	if opts.Registry == nil {
		t.Error("Options().Registry is nil")
	}
	if !opts.Signal {
		t.Error("Options().Signal should default to true")
	}
}

func TestServiceStartStop(t *testing.T) {
	svc := New(Name("startstop"), Address(":0"))
	svc.Init()

	if err := svc.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	if err := svc.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestServiceRunWithCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	svc := New(
		Name("run-cancel"),
		Address(":0"),
		HandleSignal(false),
		Context(ctx),
	)
	svc.Init()

	done := make(chan error, 1)
	go func() {
		done <- svc.Run()
	}()

	// Give the service time to start
	time.Sleep(100 * time.Millisecond)

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return after context cancel")
	}
}

func TestServiceLifecycleHooks(t *testing.T) {
	var order []string

	svc := New(
		Name("hooks"),
		Address(":0"),
		BeforeStart(func() error { order = append(order, "before-start"); return nil }),
		AfterStart(func() error { order = append(order, "after-start"); return nil }),
		BeforeStop(func() error { order = append(order, "before-stop"); return nil }),
		AfterStop(func() error { order = append(order, "after-stop"); return nil }),
	)
	svc.Init()

	if err := svc.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if err := svc.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}

	expected := []string{"before-start", "after-start", "before-stop", "after-stop"}
	if len(order) != len(expected) {
		t.Fatalf("hooks called %d times, want %d: %v", len(order), len(expected), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("hook[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestServiceHandle(t *testing.T) {
	svc := New(Name("handle"), Address(":0"))
	type Handler struct{}
	err := svc.Handle(&Handler{})
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
}

func TestGroupRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	svc1 := New(Name("g1"), Address(":0"), HandleSignal(false), Context(ctx))
	svc2 := New(Name("g2"), Address(":0"), HandleSignal(false), Context(ctx))
	g := NewGroup(svc1, svc2)

	done := make(chan error, 1)
	go func() {
		done <- g.Run()
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Group.Run() error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Group.Run() did not return after context cancel")
	}
}
