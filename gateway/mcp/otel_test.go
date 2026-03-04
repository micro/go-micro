package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v5/auth"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// newTestTP creates a TracerProvider with an in-memory exporter.
func newTestTP() (*tracetest.InMemoryExporter, trace.TracerProvider) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	return exp, tp
}

func TestOTel_SpanCreated(t *testing.T) {
	exp, tp := newTestTP()

	s := newTestServer(Options{TraceProvider: tp})
	s.tools["svc.Echo"] = &Tool{
		Name:     "svc.Echo",
		Service:  "svc",
		Endpoint: "Echo",
	}

	body, _ := json.Marshal(map[string]interface{}{
		"tool":  "svc.Echo",
		"input": map[string]interface{}{"msg": "hi"},
	})
	req := httptest.NewRequest("POST", "/mcp/call", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.handleCallTool(rec, req)

	// RPC will fail (no backend), but a span should still be created.
	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	span := spans[0]
	if span.Name != spanNameToolCall {
		t.Errorf("span name = %q, want %q", span.Name, spanNameToolCall)
	}
	if span.SpanKind != trace.SpanKindServer {
		t.Errorf("span kind = %v, want %v", span.SpanKind, trace.SpanKindServer)
	}

	// Check attributes
	assertAttr(t, span.Attributes, AttrToolName, "svc.Echo")
	assertAttr(t, span.Attributes, AttrTransport, "http")
}

func TestOTel_SpanAttributes_AuthDenied(t *testing.T) {
	exp, tp := newTestTP()

	ma := &mockAuth{
		accounts: map[string]*auth.Account{
			"tok": {ID: "user-1", Scopes: []string{"blog:read"}},
		},
	}

	s := newTestServer(Options{TraceProvider: tp, Auth: ma})
	s.tools["svc.Do"] = &Tool{
		Name:     "svc.Do",
		Service:  "svc",
		Endpoint: "Do",
		Scopes:   []string{"blog:write"},
	}

	body, _ := json.Marshal(map[string]interface{}{
		"tool":  "svc.Do",
		"input": map[string]interface{}{},
	})
	req := httptest.NewRequest("POST", "/mcp/call", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	s.handleCallTool(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected a span for denied call")
	}

	span := spans[0]
	assertAttr(t, span.Attributes, AttrAccountID, "user-1")
	assertAttrBool(t, span.Attributes, AttrAuthAllowed, false)
	assertAttr(t, span.Attributes, AttrAuthDeniedReason, "insufficient scopes")

	if span.Status.Code != codes.Error {
		t.Errorf("span status = %v, want Error", span.Status.Code)
	}
}

func TestOTel_SpanAttributes_AuthAllowed(t *testing.T) {
	exp, tp := newTestTP()

	ma := &mockAuth{
		accounts: map[string]*auth.Account{
			"tok": {ID: "user-1", Scopes: []string{"blog:write"}},
		},
	}

	s := newTestServer(Options{TraceProvider: tp, Auth: ma})
	s.tools["svc.Do"] = &Tool{
		Name:     "svc.Do",
		Service:  "svc",
		Endpoint: "Do",
		Scopes:   []string{"blog:write"},
	}

	body, _ := json.Marshal(map[string]interface{}{
		"tool":  "svc.Do",
		"input": map[string]interface{}{},
	})
	req := httptest.NewRequest("POST", "/mcp/call", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	s.handleCallTool(rec, req)

	// RPC will fail but auth should pass
	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected a span")
	}

	span := spans[0]
	assertAttr(t, span.Attributes, AttrAccountID, "user-1")
	assertAttrBool(t, span.Attributes, AttrAuthAllowed, true)

	// RPC fails, so span should have error status
	if span.Status.Code != codes.Error {
		t.Errorf("span status = %v, want Error (RPC fails with no backend)", span.Status.Code)
	}
}

func TestOTel_SpanAttributes_RateLimit(t *testing.T) {
	exp, tp := newTestTP()

	s := newTestServer(Options{
		TraceProvider: tp,
		RateLimit:     &RateLimitConfig{RequestsPerSecond: 1, Burst: 1},
	})
	s.tools["svc.Do"] = &Tool{
		Name:     "svc.Do",
		Service:  "svc",
		Endpoint: "Do",
	}
	s.limiters["svc.Do"] = newRateLimiter(1, 1)

	makeReq := func() int {
		body, _ := json.Marshal(map[string]interface{}{
			"tool":  "svc.Do",
			"input": map[string]interface{}{},
		})
		req := httptest.NewRequest("POST", "/mcp/call", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		s.handleCallTool(rec, req)
		return rec.Code
	}

	// First request passes rate limit
	makeReq()

	// Second request should be rate limited
	code := makeReq()
	if code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", code, http.StatusTooManyRequests)
	}

	spans := exp.GetSpans()
	// Find the rate-limited span
	var found bool
	for _, span := range spans {
		for _, attr := range span.Attributes {
			if string(attr.Key) == AttrRateLimited && attr.Value.AsBool() {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected a span with mcp.rate_limited=true")
	}
}

func TestOTel_NoProvider_NoSpan(t *testing.T) {
	// Without TraceProvider, tool calls should still work normally.
	s := newTestServer(Options{})
	s.tools["svc.Echo"] = &Tool{
		Name:     "svc.Echo",
		Service:  "svc",
		Endpoint: "Echo",
	}

	body, _ := json.Marshal(map[string]interface{}{
		"tool":  "svc.Echo",
		"input": map[string]interface{}{},
	})
	req := httptest.NewRequest("POST", "/mcp/call", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.handleCallTool(rec, req)

	// Should not panic or error due to missing provider.
	// RPC fails as usual.
	if rec.Code == 0 {
		t.Error("expected a response code")
	}
}

func TestOTel_TraceContextPropagation(t *testing.T) {
	exp, tp := newTestTP()

	s := newTestServer(Options{TraceProvider: tp})
	s.tools["svc.Echo"] = &Tool{
		Name:     "svc.Echo",
		Service:  "svc",
		Endpoint: "Echo",
	}

	body, _ := json.Marshal(map[string]interface{}{
		"tool":  "svc.Echo",
		"input": map[string]interface{}{},
	})
	req := httptest.NewRequest("POST", "/mcp/call", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.handleCallTool(rec, req)

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected a span")
	}

	// The span should have a valid trace ID (non-zero)
	span := spans[0]
	if !span.SpanContext.TraceID().IsValid() {
		t.Error("expected a valid OTel trace ID")
	}
	if !span.SpanContext.SpanID().IsValid() {
		t.Error("expected a valid OTel span ID")
	}
}

func TestOTel_MissingToken(t *testing.T) {
	exp, tp := newTestTP()

	ma := &mockAuth{
		accounts: map[string]*auth.Account{},
	}

	s := newTestServer(Options{TraceProvider: tp, Auth: ma})
	s.tools["svc.Do"] = &Tool{
		Name:     "svc.Do",
		Service:  "svc",
		Endpoint: "Do",
	}

	body, _ := json.Marshal(map[string]interface{}{
		"tool":  "svc.Do",
		"input": map[string]interface{}{},
	})
	req := httptest.NewRequest("POST", "/mcp/call", bytes.NewReader(body))
	// No Authorization header
	rec := httptest.NewRecorder()
	s.handleCallTool(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected a span even for missing token")
	}

	span := spans[0]
	assertAttrBool(t, span.Attributes, AttrAuthAllowed, false)
	assertAttr(t, span.Attributes, AttrAuthDeniedReason, "missing token")
}

func TestOTel_StartToolSpan_NilProvider(t *testing.T) {
	s := newTestServer(Options{})
	ctx, span := s.startToolSpan(context.Background(), "svc.Test", "http", "test-trace-id")
	defer span.End()

	// Should return a noop span, not panic
	if ctx == nil {
		t.Error("expected non-nil context")
	}
	if span == nil {
		t.Error("expected non-nil span (even if noop)")
	}
}

// --- helpers ---

func assertAttr(t *testing.T, attrs []attribute.KeyValue, key, want string) {
	t.Helper()
	for _, attr := range attrs {
		if string(attr.Key) == key {
			if got := attr.Value.AsString(); got != want {
				t.Errorf("attribute %s = %q, want %q", key, got, want)
			}
			return
		}
	}
	t.Errorf("attribute %s not found", key)
}

func assertAttrBool(t *testing.T, attrs []attribute.KeyValue, key string, want bool) {
	t.Helper()
	for _, attr := range attrs {
		if string(attr.Key) == key {
			if got := attr.Value.AsBool(); got != want {
				t.Errorf("attribute %s = %v, want %v", key, got, want)
			}
			return
		}
	}
	t.Errorf("attribute %s not found", key)
}
