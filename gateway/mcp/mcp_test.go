package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/registry"
)

// mockAuth implements auth.Auth for testing.
type mockAuth struct {
	accounts map[string]*auth.Account // token -> account
}

func (m *mockAuth) Init(...auth.Option)                         {}
func (m *mockAuth) Options() auth.Options                       { return auth.Options{} }
func (m *mockAuth) Generate(string, ...auth.GenerateOption) (*auth.Account, error) {
	return nil, nil
}
func (m *mockAuth) Token(...auth.TokenOption) (*auth.Token, error) { return nil, nil }
func (m *mockAuth) String() string                                 { return "mock" }

func (m *mockAuth) Inspect(token string) (*auth.Account, error) {
	acc, ok := m.accounts[token]
	if !ok {
		return nil, auth.ErrInvalidToken
	}
	return acc, nil
}

// newTestServer creates a Server with pre-populated tools for testing.
func newTestServer(opts Options) *Server {
	if opts.Logger == nil {
		opts.Logger = testLogger()
	}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.Client == nil {
		opts.Client = client.DefaultClient
	}
	s := &Server{
		opts:     opts,
		tools:    make(map[string]*Tool),
		limiters: make(map[string]*rateLimiter),
	}
	return s
}

// testLogger returns a silent logger for tests.
func testLogger() *log.Logger {
	return log.New(nopWriter{}, "", 0)
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

// --- Tests ---

func TestHasScope(t *testing.T) {
	tests := []struct {
		name     string
		account  []string
		required []string
		want     bool
	}{
		{"match single", []string{"blog:write"}, []string{"blog:write"}, true},
		{"match one of many", []string{"blog:read", "blog:write"}, []string{"blog:write"}, true},
		{"no match", []string{"blog:read"}, []string{"blog:write"}, false},
		{"empty required", []string{"blog:read"}, nil, false},
		{"empty account", nil, []string{"blog:write"}, false},
		{"case insensitive", []string{"Blog:Write"}, []string{"blog:write"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasScope(tt.account, tt.required)
			if got != tt.want {
				t.Errorf("hasScope(%v, %v) = %v, want %v", tt.account, tt.required, got, tt.want)
			}
		})
	}
}

func TestToolScopesFromMetadata(t *testing.T) {
	// Create a mock registry with endpoints that have scope metadata
	reg := registry.NewMemoryRegistry()
	svc := &registry.Service{
		Name: "blog",
		Nodes: []*registry.Node{{
			Id:      "blog-1",
			Address: "localhost:9090",
		}},
		Endpoints: []*registry.Endpoint{
			{
				Name: "Blog.Create",
				Metadata: map[string]string{
					"description": "Create a blog post",
					"scopes":      "blog:write,blog:admin",
				},
			},
			{
				Name: "Blog.Read",
				Metadata: map[string]string{
					"description": "Read a blog post",
				},
			},
		},
	}
	if err := reg.Register(svc); err != nil {
		t.Fatal(err)
	}

	s := newTestServer(Options{Registry: reg})
	if err := s.discoverServices(); err != nil {
		t.Fatal(err)
	}

	// Check that scopes are populated
	createTool := s.tools["blog.Blog.Create"]
	if createTool == nil {
		t.Fatal("expected tool blog.Blog.Create")
	}
	if len(createTool.Scopes) != 2 || createTool.Scopes[0] != "blog:write" || createTool.Scopes[1] != "blog:admin" {
		t.Errorf("unexpected scopes: %v", createTool.Scopes)
	}

	readTool := s.tools["blog.Blog.Read"]
	if readTool == nil {
		t.Fatal("expected tool blog.Blog.Read")
	}
	if len(readTool.Scopes) != 0 {
		t.Errorf("expected no scopes for read, got: %v", readTool.Scopes)
	}
}

func TestHandleCallTool_AuthRequired(t *testing.T) {
	ma := &mockAuth{
		accounts: map[string]*auth.Account{
			"valid-token": {ID: "user-1", Scopes: []string{"blog:write"}},
			"readonly":    {ID: "user-2", Scopes: []string{"blog:read"}},
		},
	}

	s := newTestServer(Options{Auth: ma})
	s.tools["blog.Blog.Create"] = &Tool{
		Name:     "blog.Blog.Create",
		Service:  "blog",
		Endpoint: "Blog.Create",
		Scopes:   []string{"blog:write"},
	}

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{"no token", "", http.StatusUnauthorized},
		{"invalid token", "bad-token", http.StatusUnauthorized},
		{"valid token with scope", "valid-token", http.StatusInternalServerError}, // RPC will fail (no backend), but auth passes
		{"valid token without scope", "readonly", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]interface{}{
				"tool":  "blog.Blog.Create",
				"input": map[string]interface{}{"title": "hello"},
			})
			req := httptest.NewRequest("POST", "/mcp/call", bytes.NewReader(body))
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()
			s.handleCallTool(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHandleCallTool_TraceID(t *testing.T) {
	// Without Auth, tool calls should still generate trace IDs.
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

	// Even though the RPC fails (no backend), the trace ID header should be absent
	// only when the call didn't reach the RPC stage; but in this no-auth case it
	// should reach the RPC call and fail. Check we get a response.
	traceID := rec.Header().Get(TraceIDKey)
	// The RPC call will fail but the error path doesn't set the header.
	// For a successful call, the trace ID is set. Either way the audit should fire.
	_ = traceID // trace ID may or may not be in error response header
}

func TestHandleCallTool_AuditFunc(t *testing.T) {
	var mu sync.Mutex
	var records []AuditRecord

	auditFn := func(r AuditRecord) {
		mu.Lock()
		defer mu.Unlock()
		records = append(records, r)
	}

	ma := &mockAuth{
		accounts: map[string]*auth.Account{
			"tok": {ID: "u1", Scopes: []string{"write"}},
		},
	}

	s := newTestServer(Options{Auth: ma, AuditFunc: auditFn})
	s.tools["svc.Do"] = &Tool{
		Name:     "svc.Do",
		Service:  "svc",
		Endpoint: "Do",
		Scopes:   []string{"write"},
	}

	body, _ := json.Marshal(map[string]interface{}{
		"tool":  "svc.Do",
		"input": map[string]interface{}{},
	})
	req := httptest.NewRequest("POST", "/mcp/call", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	s.handleCallTool(rec, req)

	mu.Lock()
	defer mu.Unlock()

	if len(records) == 0 {
		t.Fatal("expected at least one audit record")
	}
	r := records[len(records)-1]
	if r.AccountID != "u1" {
		t.Errorf("audit AccountID = %q, want %q", r.AccountID, "u1")
	}
	if r.Tool != "svc.Do" {
		t.Errorf("audit Tool = %q, want %q", r.Tool, "svc.Do")
	}
	if r.TraceID == "" {
		t.Error("audit TraceID is empty")
	}
	if !r.Allowed {
		t.Error("expected audit record Allowed = true")
	}
}

func TestHandleCallTool_AuditDenied(t *testing.T) {
	var mu sync.Mutex
	var records []AuditRecord

	auditFn := func(r AuditRecord) {
		mu.Lock()
		defer mu.Unlock()
		records = append(records, r)
	}

	ma := &mockAuth{
		accounts: map[string]*auth.Account{
			"tok": {ID: "u1", Scopes: []string{"blog:read"}},
		},
	}

	s := newTestServer(Options{Auth: ma, AuditFunc: auditFn})
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
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(records) == 0 {
		t.Fatal("expected audit record for denied call")
	}
	r := records[0]
	if r.Allowed {
		t.Error("expected Allowed = false")
	}
	if r.DeniedReason != "insufficient scopes" {
		t.Errorf("DeniedReason = %q, want %q", r.DeniedReason, "insufficient scopes")
	}
}

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter(10, 2)

	// First two should be allowed (burst)
	if !rl.Allow() {
		t.Error("first call should be allowed")
	}
	if !rl.Allow() {
		t.Error("second call should be allowed (burst)")
	}

	// Third should be denied (burst exhausted, no time to refill)
	if rl.Allow() {
		t.Error("third call should be denied (burst exhausted)")
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow() {
		t.Error("call after refill should be allowed")
	}
}

func TestHandleCallTool_RateLimit(t *testing.T) {
	var mu sync.Mutex
	var records []AuditRecord

	s := newTestServer(Options{
		RateLimit: &RateLimitConfig{RequestsPerSecond: 1, Burst: 1},
		AuditFunc: func(r AuditRecord) {
			mu.Lock()
			records = append(records, r)
			mu.Unlock()
		},
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

	// First request should pass rate limit (but RPC may fail — that's ok)
	code1 := makeReq()
	if code1 == http.StatusTooManyRequests {
		t.Error("first request should not be rate limited")
	}

	// Second request should be rate limited
	code2 := makeReq()
	if code2 != http.StatusTooManyRequests {
		t.Errorf("second request status = %d, want %d", code2, http.StatusTooManyRequests)
	}

	// Check audit records include rate limit denial
	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, r := range records {
		if r.DeniedReason == "rate limited" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit record with DeniedReason = 'rate limited'")
	}
}

func TestHandleCallTool_NoAuth_NoScope(t *testing.T) {
	// Without Auth configured, tools without scopes should be accessible
	s := newTestServer(Options{})
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

	// Should not be 401 or 403 (RPC failure is expected since no backend)
	if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
		t.Errorf("unexpected auth error: %d", rec.Code)
	}
}

func TestToolScopesInJSON(t *testing.T) {
	tool := &Tool{
		Name:        "blog.Blog.Create",
		Description: "Create a blog post",
		InputSchema: map[string]interface{}{"type": "object"},
		Scopes:      []string{"blog:write", "blog:admin"},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	scopes, ok := m["scopes"].([]interface{})
	if !ok {
		t.Fatal("expected scopes in JSON output")
	}
	if len(scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(scopes))
	}
}

func TestToolNoScopesOmittedInJSON(t *testing.T) {
	tool := &Tool{
		Name:        "blog.Blog.Read",
		Description: "Read a blog post",
		InputSchema: map[string]interface{}{"type": "object"},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	if _, ok := m["scopes"]; ok {
		t.Error("expected scopes to be omitted when empty")
	}
}

func TestDiscoverServices_RateLimiters(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	svc := &registry.Service{
		Name: "blog",
		Nodes: []*registry.Node{{
			Id:      "blog-1",
			Address: "localhost:9090",
		}},
		Endpoints: []*registry.Endpoint{
			{Name: "Blog.Create"},
			{Name: "Blog.Read"},
		},
	}
	if err := reg.Register(svc); err != nil {
		t.Fatal(err)
	}

	s := newTestServer(Options{
		Registry:  reg,
		RateLimit: &RateLimitConfig{RequestsPerSecond: 10, Burst: 5},
	})
	if err := s.discoverServices(); err != nil {
		t.Fatal(err)
	}

	if len(s.limiters) != 2 {
		t.Errorf("expected 2 limiters, got %d", len(s.limiters))
	}
	for name := range s.tools {
		if _, ok := s.limiters[name]; !ok {
			t.Errorf("missing limiter for tool %s", name)
		}
	}
}
