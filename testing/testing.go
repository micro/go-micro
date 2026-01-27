// Package testing provides utilities for testing micro services.
//
// Due to go-micro's global defaults, running multiple services in one process
// requires careful isolation. This package provides helpers for the common case
// of testing a single service.
//
// Basic usage:
//
//	func TestUserService(t *testing.T) {
//	    h := testing.NewHarness(t)
//	    defer h.Stop()
//	
//	    // Register your service handler
//	    h.Register(new(UsersHandler))
//	
//	    // Start the harness
//	    h.Start()
//	
//	    // Call the service
//	    var rsp UserResponse
//	    err := h.Call("Users.Create", &CreateRequest{Name: "Alice"}, &rsp)
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//	}
package testing

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/transport"
)

// Harness provides an in-process test environment for a micro service
type Harness struct {
	t         *testing.T
	name      string
	handler   interface{}
	registry  registry.Registry
	transport transport.Transport
	broker    broker.Broker
	server    server.Server
	client    client.Client
	started   bool
	mu        sync.Mutex
}

// NewHarness creates a new test harness
func NewHarness(t *testing.T) *Harness {
	// Create isolated instances for testing
	reg := registry.NewMemoryRegistry()
	tr := transport.NewHTTPTransport()
	br := broker.NewMemoryBroker()

	return &Harness{
		t:         t,
		name:      "test",
		registry:  reg,
		transport: tr,
		broker:    br,
	}
}

// Name sets the service name (default: "test")
func (h *Harness) Name(name string) *Harness {
	h.name = name
	return h
}

// Register sets the handler for the service
func (h *Harness) Register(handler interface{}) *Harness {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		h.t.Fatal("cannot register handler after Start()")
	}

	h.handler = handler
	return h
}

// Start starts the service
func (h *Harness) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		return
	}

	if h.handler == nil {
		h.t.Fatal("no handler registered, call Register() first")
	}

	// Connect broker
	if err := h.broker.Connect(); err != nil {
		h.t.Fatalf("failed to connect broker: %v", err)
	}

	// Create server with isolated transport
	h.server = server.NewServer(
		server.Name(h.name),
		server.Registry(h.registry),
		server.Transport(h.transport),
		server.Broker(h.broker),
		server.Address("127.0.0.1:0"),
	)

	// Register handler
	if err := h.server.Handle(h.server.NewHandler(h.handler)); err != nil {
		h.t.Fatalf("failed to register handler: %v", err)
	}

	// Start server
	if err := h.server.Start(); err != nil {
		h.t.Fatalf("failed to start server: %v", err)
	}

	// Create client with same registry/transport
	h.client = client.NewClient(
		client.Registry(h.registry),
		client.Transport(h.transport),
		client.Broker(h.broker),
		client.RequestTimeout(5*time.Second),
	)

	// Wait for registration
	h.waitForService()

	h.started = true
}

func (h *Harness) waitForService() {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		services, err := h.registry.GetService(h.name)
		if err == nil && len(services) > 0 && len(services[0].Nodes) > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	h.t.Fatalf("service %s did not register in time", h.name)
}

// Stop stops the service
func (h *Harness) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.server != nil {
		h.server.Stop()
	}
	if h.broker != nil {
		h.broker.Disconnect()
	}

	h.started = false
}

// Call invokes a service method
func (h *Harness) Call(endpoint string, req, rsp interface{}) error {
	return h.CallContext(context.Background(), endpoint, req, rsp)
}

// CallContext invokes a service method with context
func (h *Harness) CallContext(ctx context.Context, endpoint string, req, rsp interface{}) error {
	if !h.started {
		return fmt.Errorf("harness not started, call Start() first")
	}

	request := h.client.NewRequest(h.name, endpoint, req)
	return h.client.Call(ctx, request, rsp)
}

// Client returns the test client for advanced usage
func (h *Harness) Client() client.Client {
	return h.client
}

// Server returns the test server for advanced usage  
func (h *Harness) Server() server.Server {
	return h.server
}

// Registry returns the test registry for advanced usage
func (h *Harness) Registry() registry.Registry {
	return h.registry
}

// --- Assertions ---

// AssertServiceRunning checks that the service is registered
func (h *Harness) AssertServiceRunning() {
	h.t.Helper()

	services, err := h.registry.GetService(h.name)
	if err != nil {
		h.t.Errorf("service %s not found: %v", h.name, err)
		return
	}
	if len(services) == 0 || len(services[0].Nodes) == 0 {
		h.t.Errorf("service %s has no running instances", h.name)
	}
}

// AssertCallSucceeds checks that a call succeeds
func (h *Harness) AssertCallSucceeds(endpoint string, req, rsp interface{}) {
	h.t.Helper()

	if err := h.Call(endpoint, req, rsp); err != nil {
		h.t.Errorf("call %s failed: %v", endpoint, err)
	}
}

// AssertCallFails checks that a call fails
func (h *Harness) AssertCallFails(endpoint string, req, rsp interface{}) {
	h.t.Helper()

	if err := h.Call(endpoint, req, rsp); err == nil {
		h.t.Errorf("expected call %s to fail, but it succeeded", endpoint)
	}
}
