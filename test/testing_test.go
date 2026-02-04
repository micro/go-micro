package test

import (
	"context"
	"testing"
)

// Simple test handler
type GreeterHandler struct{}

type HelloRequest struct {
	Name string `json:"name"`
}

type HelloResponse struct {
	Message string `json:"message"`
}

func (g *GreeterHandler) Hello(ctx context.Context, req *HelloRequest, rsp *HelloResponse) error {
	rsp.Message = "Hello " + req.Name
	return nil
}

func TestHarnessBasic(t *testing.T) {
	h := NewHarness(t)
	defer h.Stop()

	h.Name("greeter").Register(new(GreeterHandler))
	h.Start()

	// Check service is running
	h.AssertServiceRunning()

	// Make a call
	var rsp HelloResponse
	err := h.Call("GreeterHandler.Hello", &HelloRequest{Name: "World"}, &rsp)
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	if rsp.Message != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", rsp.Message)
	}
}

func TestHarnessCallBeforeStart(t *testing.T) {
	h := NewHarness(t)
	defer h.Stop()

	h.Register(new(GreeterHandler))
	// Don't call Start()

	var rsp HelloResponse
	err := h.Call("GreeterHandler.Hello", &HelloRequest{Name: "World"}, &rsp)
	if err == nil {
		t.Error("expected error when calling before Start()")
	}
}

func TestHarnessAssertCallSucceeds(t *testing.T) {
	h := NewHarness(t)
	defer h.Stop()

	h.Name("greeter").Register(new(GreeterHandler))
	h.Start()

	var rsp HelloResponse
	h.AssertCallSucceeds("GreeterHandler.Hello", &HelloRequest{Name: "Test"}, &rsp)

	if rsp.Message != "Hello Test" {
		t.Errorf("expected 'Hello Test', got '%s'", rsp.Message)
	}
}

func TestHarnessClientAndServer(t *testing.T) {
	h := NewHarness(t)
	defer h.Stop()

	h.Name("greeter").Register(new(GreeterHandler))
	h.Start()

	// Check we can access client and server
	if h.Client() == nil {
		t.Fatal("client is nil")
	}
	if h.Server() == nil {
		t.Fatal("server is nil")
	}
	if h.Registry() == nil {
		t.Fatal("registry is nil")
	}
}

func TestHarnessWithContext(t *testing.T) {
	h := NewHarness(t)
	defer h.Stop()

	h.Name("greeter").Register(new(GreeterHandler))
	h.Start()

	ctx := context.Background()
	var rsp HelloResponse
	err := h.CallContext(ctx, "GreeterHandler.Hello", &HelloRequest{Name: "Context"}, &rsp)
	if err != nil {
		t.Fatalf("call with context failed: %v", err)
	}

	if rsp.Message != "Hello Context" {
		t.Errorf("expected 'Hello Context', got '%s'", rsp.Message)
	}
}
