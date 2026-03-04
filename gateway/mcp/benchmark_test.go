package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/registry"
)

// benchServer creates a Server with N pre-populated tools.
func benchServer(n int, opts Options) *Server {
	if opts.Logger == nil {
		opts.Logger = log.New(log.Writer(), "", 0)
	}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.Client == nil {
		opts.Client = client.DefaultClient
	}
	if opts.Registry == nil {
		opts.Registry = registry.DefaultRegistry
	}

	s := &Server{
		opts:     opts,
		tools:    make(map[string]*Tool, n),
		limiters: make(map[string]*rateLimiter),
	}

	for i := 0; i < n; i++ {
		name := toolName(i)
		s.tools[name] = &Tool{
			Name:        name,
			Description: "Benchmark tool " + name,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Resource identifier",
					},
				},
				"required": []interface{}{"id"},
			},
			Service:  "bench",
			Endpoint: "Handler.Method",
		}
	}

	return s
}

func toolName(i int) string {
	return "bench.Handler.Method" + string(rune('A'+i%26))
}

// --- Benchmarks ---

// BenchmarkListTools measures tool listing throughput.
// This is the most common MCP operation — agents call it on every session start.
func BenchmarkListTools(b *testing.B) {
	for _, numTools := range []int{10, 50, 100} {
		b.Run(toolCountLabel(numTools), func(b *testing.B) {
			s := benchServer(numTools, Options{})
			req := httptest.NewRequest("GET", "/mcp/tools", nil)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				w := httptest.NewRecorder()
				s.handleListTools(w, req)
				if w.Code != http.StatusOK {
					b.Fatalf("unexpected status %d", w.Code)
				}
			}
		})
	}
}

// BenchmarkListToolsParallel measures concurrent tool listing.
func BenchmarkListToolsParallel(b *testing.B) {
	s := benchServer(50, Options{})
	req := httptest.NewRequest("GET", "/mcp/tools", nil)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			s.handleListTools(w, req)
		}
	})
}

// BenchmarkToolLookup measures tool name resolution from the tools map.
func BenchmarkToolLookup(b *testing.B) {
	for _, numTools := range []int{10, 50, 100, 500} {
		b.Run(toolCountLabel(numTools), func(b *testing.B) {
			s := benchServer(numTools, Options{})
			name := toolName(numTools / 2) // look up a tool in the middle

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				s.toolsMu.RLock()
				_, ok := s.tools[name]
				s.toolsMu.RUnlock()
				if !ok {
					b.Fatal("tool not found")
				}
			}
		})
	}
}

// BenchmarkAuthInspect measures auth token inspection overhead.
func BenchmarkAuthInspect(b *testing.B) {
	ma := &mockAuth{
		accounts: map[string]*auth.Account{
			"valid-token": {
				ID:     "bench-user",
				Scopes: []string{"read", "write"},
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		acc, err := ma.Inspect("valid-token")
		if err != nil || acc.ID != "bench-user" {
			b.Fatal("unexpected result")
		}
	}
}

// BenchmarkScopeCheck measures scope validation overhead per tool call.
func BenchmarkScopeCheck(b *testing.B) {
	accountScopes := []string{"users:read", "users:write", "orders:read", "admin"}
	requiredScopes := []string{"users:write"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hasScope(accountScopes, requiredScopes)
	}
}

// BenchmarkAuditRecord measures audit record creation overhead.
func BenchmarkAuditRecord(b *testing.B) {
	var records int
	s := benchServer(10, Options{
		AuditFunc: func(r AuditRecord) {
			records++
		},
	})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.opts.AuditFunc(AuditRecord{
			TraceID:   "trace-123",
			Tool:      "bench.Handler.MethodA",
			AccountID: "user-1",
			Allowed:   true,
		})
	}
}

// BenchmarkRateLimiter measures rate limiter check overhead.
func BenchmarkRateLimiter(b *testing.B) {
	s := benchServer(10, Options{
		RateLimit: &RateLimitConfig{
			RequestsPerSecond: 1000000, // Very high so it doesn't block
			Burst:             1000000,
		},
	})
	// Initialize limiters for tools
	for name := range s.tools {
		s.limiters[name] = newRateLimiter(s.opts.RateLimit.RequestsPerSecond, s.opts.RateLimit.Burst)
	}
	name := toolName(0)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.limitersMu.RLock()
		l := s.limiters[name]
		s.limitersMu.RUnlock()
		l.Allow()
	}
}

// BenchmarkJSONEncodeTool measures JSON serialization of tool definitions.
func BenchmarkJSONEncodeTool(b *testing.B) {
	tool := &Tool{
		Name:        "myservice.Users.GetUser",
		Description: "Retrieve a user by their unique ID. Returns the full profile.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"description": "User ID in UUID format",
				},
			},
			"required": []interface{}{"id"},
		},
		Scopes:   []string{"users:read"},
		Service:  "myservice",
		Endpoint: "Users.GetUser",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(tool)
	}
}

// BenchmarkJSONDecodeCallRequest measures parsing of incoming tool call requests.
func BenchmarkJSONDecodeCallRequest(b *testing.B) {
	body := []byte(`{"tool":"myservice.Users.GetUser","arguments":{"id":"user-123"}}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var req struct {
			Tool      string                 `json:"tool"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		json.Unmarshal(body, &req)
	}
}

// --- Helpers ---

func toolCountLabel(n int) string {
	switch {
	case n >= 500:
		return "500_tools"
	case n >= 100:
		return "100_tools"
	case n >= 50:
		return "50_tools"
	default:
		return "10_tools"
	}
}

