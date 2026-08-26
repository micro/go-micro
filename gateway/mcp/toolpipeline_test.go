package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go-micro.dev/v6/auth"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/codec/bytes"
)

// fakeCallClient2 overrides Call to return canned data or an error; NewRequest
// is promoted from the embedded real client.
type fakeCallClient2 struct {
	client.Client
	data []byte
	err  error
}

func (f *fakeCallClient2) Call(_ context.Context, _ client.Request, rsp interface{}, _ ...client.CallOption) error {
	if f.err != nil {
		return f.err
	}
	if r, ok := rsp.(*bytes.Frame); ok {
		r.Data = f.data
	}
	return nil
}

// --- Tracer bullet: basic RPC call through invokeToolCore ---

func TestInvokeToolCore_BasicRPCCall(t *testing.T) {
	s := newTestServer(Options{
		Client: &fakeCallClient2{Client: client.DefaultClient, data: []byte(`{"ok":true}`)},
	})
	s.tools["svc.Do"] = &Tool{Name: "svc.Do", Service: "svc", Endpoint: "Do"}

	payload, traceID, _, err := s.invokeToolCore(toolCallRequest{
		ToolName:  "svc.Do",
		Input:     map[string]interface{}{"x": 1},
		BaseCtx:   context.Background(),
		Transport: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if traceID == "" {
		t.Error("expected non-empty traceID")
	}
	var got map[string]interface{}
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("payload not JSON: %v", err)
	}
	if got["ok"] != true {
		t.Errorf("payload = %v, want ok=true", got)
	}
}

// --- Auth: missing token → 401 ---

func TestInvokeToolCore_MissingToken(t *testing.T) {
	ma := &mockAuth{accounts: map[string]*auth.Account{}}
	s := newTestServer(Options{Auth: ma})
	s.tools["svc.Do"] = &Tool{Name: "svc.Do", Service: "svc", Endpoint: "Do"}

	_, _, _, err := s.invokeToolCore(toolCallRequest{
		ToolName:  "svc.Do",
		Input:     map[string]interface{}{},
		Token:     "",
		BaseCtx:   context.Background(),
		Transport: "test",
	})
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	te, ok := err.(*toolError)
	if !ok {
		t.Fatalf("expected *toolError, got %T: %v", err, err)
	}
	if te.status != 401 {
		t.Errorf("status = %d, want 401", te.status)
	}
}

// --- Auth: invalid token → 401 ---

func TestInvokeToolCore_InvalidToken(t *testing.T) {
	ma := &mockAuth{accounts: map[string]*auth.Account{}}
	s := newTestServer(Options{Auth: ma})
	s.tools["svc.Do"] = &Tool{Name: "svc.Do", Service: "svc", Endpoint: "Do"}

	_, _, _, err := s.invokeToolCore(toolCallRequest{
		ToolName:  "svc.Do",
		Input:     map[string]interface{}{},
		Token:     "bad-token",
		BaseCtx:   context.Background(),
		Transport: "test",
	})
	te, ok := err.(*toolError)
	if !ok {
		t.Fatalf("expected *toolError, got %T: %v", err, err)
	}
	if te.status != 401 {
		t.Errorf("status = %d, want 401", te.status)
	}
}

// --- Scope check: insufficient scopes → 403 ---

func TestInvokeToolCore_InsufficientScopes(t *testing.T) {
	ma := &mockAuth{accounts: map[string]*auth.Account{
		"tok": {ID: "user-1", Scopes: []string{"read"}},
	}}
	s := newTestServer(Options{Auth: ma})
	s.tools["svc.Do"] = &Tool{Name: "svc.Do", Service: "svc", Endpoint: "Do", Scopes: []string{"write"}}

	_, _, _, err := s.invokeToolCore(toolCallRequest{
		ToolName:  "svc.Do",
		Input:     map[string]interface{}{},
		Token:     "tok",
		BaseCtx:   context.Background(),
		Transport: "test",
	})
	te, ok := err.(*toolError)
	if !ok {
		t.Fatalf("expected *toolError, got %T: %v", err, err)
	}
	if te.status != 403 {
		t.Errorf("status = %d, want 403", te.status)
	}
}

// --- Rate limit: exceeded → 429 ---

func TestInvokeToolCore_RateLimitExceeded(t *testing.T) {
	s := newTestServer(Options{
		RateLimit: &RateLimitConfig{RequestsPerSecond: 0.001, Burst: 1},
	})
	s.tools["svc.Do"] = &Tool{Name: "svc.Do", Service: "svc", Endpoint: "Do"}
	s.limiters["svc.Do"] = newRateLimiter(0.001, 1)
	// Exhaust the burst.
	s.allowRate("svc.Do")

	_, _, _, err := s.invokeToolCore(toolCallRequest{
		ToolName:  "svc.Do",
		Input:     map[string]interface{}{},
		BaseCtx:   context.Background(),
		Transport: "test",
	})
	te, ok := err.(*toolError)
	if !ok {
		t.Fatalf("expected *toolError, got %T: %v", err, err)
	}
	if te.status != 429 {
		t.Errorf("status = %d, want 429", te.status)
	}
}

// --- Circuit breaker: open → 503 ---

func TestInvokeToolCore_CircuitBreakerOpen(t *testing.T) {
	s := newTestServer(Options{
		CircuitBreaker: &CircuitBreakerConfig{MaxFailures: 1},
	})
	s.tools["svc.Do"] = &Tool{Name: "svc.Do", Service: "svc", Endpoint: "Do"}
	s.breakers = make(map[string]*circuitBreaker)
	// Force the breaker open.
	cb := newCircuitBreaker(CircuitBreakerConfig{MaxFailures: 1})
	cb.RecordFailure()
	s.breakers["svc.Do"] = cb

	_, _, _, err := s.invokeToolCore(toolCallRequest{
		ToolName:  "svc.Do",
		Input:     map[string]interface{}{},
		BaseCtx:   context.Background(),
		Transport: "test",
	})
	te, ok := err.(*toolError)
	if !ok {
		t.Fatalf("expected *toolError, got %T: %v", err, err)
	}
	if te.status != 503 {
		t.Errorf("status = %d, want 503", te.status)
	}
}

// --- Framework tool: handler called directly ---

func TestInvokeToolCore_FrameworkTool(t *testing.T) {
	s := newTestServer(Options{})
	called := false
	s.tools["fw.Summ"] = &Tool{
		Name: "fw.Summ",
		Handler: func(input map[string]interface{}) (interface{}, error) {
			called = true
			return map[string]interface{}{"sum": 42}, nil
		},
	}

	payload, _, _, err := s.invokeToolCore(toolCallRequest{
		ToolName:  "fw.Summ",
		Input:     map[string]interface{}{},
		BaseCtx:   context.Background(),
		Transport: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("framework handler was not called")
	}
	var got map[string]interface{}
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("payload not JSON: %v", err)
	}
	if got["sum"] != float64(42) {
		t.Errorf("payload = %v, want sum=42", got)
	}
}

// --- Framework tool: handler error → 500 ---

func TestInvokeToolCore_FrameworkToolError(t *testing.T) {
	s := newTestServer(Options{})
	s.tools["fw.Bomb"] = &Tool{
		Name: "fw.Bomb",
		Handler: func(input map[string]interface{}) (interface{}, error) {
			return nil, errors.New("kaboom")
		},
	}

	_, _, _, err := s.invokeToolCore(toolCallRequest{
		ToolName:  "fw.Bomb",
		Input:     map[string]interface{}{},
		BaseCtx:   context.Background(),
		Transport: "test",
	})
	te, ok := err.(*toolError)
	if !ok {
		t.Fatalf("expected *toolError, got %T: %v", err, err)
	}
	if te.status != 500 {
		t.Errorf("status = %d, want 500", te.status)
	}
}

// --- Tool not found → 404 ---

func TestInvokeToolCore_ToolNotFound(t *testing.T) {
	s := newTestServer(Options{})

	_, _, _, err := s.invokeToolCore(toolCallRequest{
		ToolName:  "no.such.Tool",
		Input:     map[string]interface{}{},
		BaseCtx:   context.Background(),
		Transport: "test",
	})
	te, ok := err.(*toolError)
	if !ok {
		t.Fatalf("expected *toolError, got %T: %v", err, err)
	}
	if te.status != 404 {
		t.Errorf("status = %d, want 404", te.status)
	}
}

// --- RPC failure records circuit breaker failure ---

func TestInvokeToolCore_CircuitRecordsFailure(t *testing.T) {
	s := newTestServer(Options{
		CircuitBreaker: &CircuitBreakerConfig{MaxFailures: 2},
		Client:         &fakeCallClient2{Client: client.DefaultClient, err: errors.New("down")},
	})
	s.tools["svc.Do"] = &Tool{Name: "svc.Do", Service: "svc", Endpoint: "Do"}
	s.breakers = make(map[string]*circuitBreaker)
	s.breakers["svc.Do"] = newCircuitBreaker(CircuitBreakerConfig{MaxFailures: 2})

	// Call twice — should trip the breaker.
	for i := 0; i < 2; i++ {
		s.invokeToolCore(toolCallRequest{
			ToolName:  "svc.Do",
			Input:     map[string]interface{}{},
			BaseCtx:   context.Background(),
			Transport: "test",
		})
	}

	// Third call should be rejected by the open breaker.
	_, _, _, err := s.invokeToolCore(toolCallRequest{
		ToolName:  "svc.Do",
		Input:     map[string]interface{}{},
		BaseCtx:   context.Background(),
		Transport: "test",
	})
	te, ok := err.(*toolError)
	if !ok {
		t.Fatalf("expected *toolError, got %T: %v", err, err)
	}
	if te.status != 503 {
		t.Errorf("status = %d, want 503 (breaker should be open)", te.status)
	}
}
