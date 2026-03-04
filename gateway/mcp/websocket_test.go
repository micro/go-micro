package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go-micro.dev/v5/auth"

	"github.com/gorilla/websocket"
)

// wsDialer creates a WebSocket connection to the given test server URL.
func wsDialer(t *testing.T, url string, headers http.Header) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(url, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// sendJSONRPC sends a JSON-RPC request and reads the response.
func sendJSONRPC(t *testing.T, conn *websocket.Conn, method string, id interface{}, params interface{}) JSONRPCResponse {
	t.Helper()
	raw, _ := json.Marshal(params)
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  raw,
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var resp JSONRPCResponse
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}
	return resp
}

func newWSTestServer(t *testing.T, opts Options) (*Server, *httptest.Server) {
	t.Helper()
	s := newTestServer(opts)
	ws := NewWebSocketTransport(s)

	mux := http.NewServeMux()
	mux.Handle("/mcp/ws", ws)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return s, ts
}

func TestWebSocket_Initialize(t *testing.T) {
	_, ts := newWSTestServer(t, Options{})
	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)

	resp := sendJSONRPC(t, conn, "initialize", 1, nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion = %v, want 2024-11-05", result["protocolVersion"])
	}
}

func TestWebSocket_ToolsList(t *testing.T) {
	s, ts := newWSTestServer(t, Options{})
	s.tools["svc.Echo"] = &Tool{
		Name:        "svc.Echo",
		Description: "Echo a message",
		InputSchema: map[string]interface{}{"type": "object"},
		Service:     "svc",
		Endpoint:    "Echo",
	}

	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)
	resp := sendJSONRPC(t, conn, "tools/list", 1, nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, _ := resp.Result.(map[string]interface{})
	tools, _ := result["tools"].([]interface{})
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
}

func TestWebSocket_ToolsCall_NoAuth(t *testing.T) {
	s, ts := newWSTestServer(t, Options{})
	s.tools["svc.Echo"] = &Tool{
		Name:     "svc.Echo",
		Service:  "svc",
		Endpoint: "Echo",
	}

	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)
	resp := sendJSONRPC(t, conn, "tools/call", 1, map[string]interface{}{
		"name":      "svc.Echo",
		"arguments": map[string]interface{}{"msg": "hi"},
	})

	// RPC will fail (no backend), but auth should pass (no auth configured)
	if resp.Error == nil {
		t.Fatal("expected RPC error (no backend)")
	}
	if resp.Error.Code != InternalError {
		t.Errorf("error code = %d, want %d", resp.Error.Code, InternalError)
	}
}

func TestWebSocket_ToolsCall_AuthRequired(t *testing.T) {
	ma := &mockAuth{
		accounts: map[string]*auth.Account{
			"valid-token": {ID: "user-1", Scopes: []string{"blog:write"}},
		},
	}

	s, ts := newWSTestServer(t, Options{Auth: ma})
	s.tools["svc.Do"] = &Tool{
		Name:     "svc.Do",
		Service:  "svc",
		Endpoint: "Do",
		Scopes:   []string{"blog:write"},
	}

	t.Run("missing token", func(t *testing.T) {
		conn := wsDialer(t, ts.URL+"/mcp/ws", nil)
		resp := sendJSONRPC(t, conn, "tools/call", 1, map[string]interface{}{
			"name":      "svc.Do",
			"arguments": map[string]interface{}{},
		})
		if resp.Error == nil || resp.Error.Message != "Unauthorized" {
			t.Errorf("expected Unauthorized, got %+v", resp.Error)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		conn := wsDialer(t, ts.URL+"/mcp/ws", nil)
		resp := sendJSONRPC(t, conn, "tools/call", 1, map[string]interface{}{
			"name":      "svc.Do",
			"arguments": map[string]interface{}{},
			"_token":    "bad-token",
		})
		if resp.Error == nil || resp.Error.Message != "Unauthorized" {
			t.Errorf("expected Unauthorized, got %+v", resp.Error)
		}
	})

	t.Run("valid token via param", func(t *testing.T) {
		conn := wsDialer(t, ts.URL+"/mcp/ws", nil)
		resp := sendJSONRPC(t, conn, "tools/call", 1, map[string]interface{}{
			"name":      "svc.Do",
			"arguments": map[string]interface{}{},
			"_token":    "valid-token",
		})
		// Auth passes, RPC fails (no backend)
		if resp.Error == nil {
			t.Fatal("expected RPC error")
		}
		if resp.Error.Code != InternalError {
			t.Errorf("error code = %d, want %d (RPC fail, not auth fail)", resp.Error.Code, InternalError)
		}
	})

	t.Run("valid token via header", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer valid-token")
		conn := wsDialer(t, ts.URL+"/mcp/ws", headers)
		resp := sendJSONRPC(t, conn, "tools/call", 1, map[string]interface{}{
			"name":      "svc.Do",
			"arguments": map[string]interface{}{},
		})
		// Auth passes via connection-level header, RPC fails (no backend)
		if resp.Error == nil {
			t.Fatal("expected RPC error")
		}
		if resp.Error.Code != InternalError {
			t.Errorf("error code = %d, want %d (RPC fail, not auth fail)", resp.Error.Code, InternalError)
		}
	})
}

func TestWebSocket_ToolsCall_InsufficientScopes(t *testing.T) {
	ma := &mockAuth{
		accounts: map[string]*auth.Account{
			"readonly": {ID: "user-2", Scopes: []string{"blog:read"}},
		},
	}

	s, ts := newWSTestServer(t, Options{Auth: ma})
	s.tools["svc.Do"] = &Tool{
		Name:     "svc.Do",
		Service:  "svc",
		Endpoint: "Do",
		Scopes:   []string{"blog:write"},
	}

	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)
	resp := sendJSONRPC(t, conn, "tools/call", 1, map[string]interface{}{
		"name":      "svc.Do",
		"arguments": map[string]interface{}{},
		"_token":    "readonly",
	})
	if resp.Error == nil || resp.Error.Message != "Forbidden" {
		t.Errorf("expected Forbidden, got %+v", resp.Error)
	}
}

func TestWebSocket_ToolsCall_Audit(t *testing.T) {
	var mu sync.Mutex
	var records []AuditRecord

	s, ts := newWSTestServer(t, Options{
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

	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)
	sendJSONRPC(t, conn, "tools/call", 1, map[string]interface{}{
		"name":      "svc.Do",
		"arguments": map[string]interface{}{},
	})

	mu.Lock()
	defer mu.Unlock()

	if len(records) == 0 {
		t.Fatal("expected audit record")
	}
	r := records[len(records)-1]
	if r.Tool != "svc.Do" {
		t.Errorf("audit Tool = %q, want %q", r.Tool, "svc.Do")
	}
	if r.TraceID == "" {
		t.Error("audit TraceID is empty")
	}
}

func TestWebSocket_RateLimit(t *testing.T) {
	s, ts := newWSTestServer(t, Options{
		RateLimit: &RateLimitConfig{RequestsPerSecond: 1, Burst: 1},
	})
	s.tools["svc.Do"] = &Tool{
		Name:     "svc.Do",
		Service:  "svc",
		Endpoint: "Do",
	}
	s.limiters["svc.Do"] = newRateLimiter(1, 1)

	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)

	params := map[string]interface{}{
		"name":      "svc.Do",
		"arguments": map[string]interface{}{},
	}

	// First request passes rate limit (RPC may fail, that's ok)
	resp1 := sendJSONRPC(t, conn, "tools/call", 1, params)
	if resp1.Error != nil && resp1.Error.Message == "Rate limit exceeded" {
		t.Error("first request should not be rate limited")
	}

	// Second request should be rate limited
	resp2 := sendJSONRPC(t, conn, "tools/call", 2, params)
	if resp2.Error == nil || resp2.Error.Message != "Rate limit exceeded" {
		t.Errorf("expected rate limit error, got %+v", resp2.Error)
	}
}

func TestWebSocket_MethodNotFound(t *testing.T) {
	_, ts := newWSTestServer(t, Options{})
	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)

	resp := sendJSONRPC(t, conn, "nonexistent/method", 1, nil)
	if resp.Error == nil || resp.Error.Code != MethodNotFound {
		t.Errorf("expected MethodNotFound, got %+v", resp.Error)
	}
}

func TestWebSocket_ToolNotFound(t *testing.T) {
	_, ts := newWSTestServer(t, Options{})
	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)

	resp := sendJSONRPC(t, conn, "tools/call", 1, map[string]interface{}{
		"name":      "nonexistent.Tool",
		"arguments": map[string]interface{}{},
	})
	if resp.Error == nil || resp.Error.Message != "Tool not found" {
		t.Errorf("expected Tool not found, got %+v", resp.Error)
	}
}

func TestWebSocket_MultipleConcurrentRequests(t *testing.T) {
	s, ts := newWSTestServer(t, Options{})
	s.tools["svc.Echo"] = &Tool{
		Name:     "svc.Echo",
		Service:  "svc",
		Endpoint: "Echo",
	}

	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)

	// Send multiple requests sequentially (gorilla client doesn't allow
	// concurrent writes), but the server handles them concurrently.
	const n = 5
	for i := 0; i < n; i++ {
		raw, _ := json.Marshal(map[string]interface{}{
			"name":      "svc.Echo",
			"arguments": map[string]interface{}{},
		})
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      i + 1,
			Method:  "tools/call",
			Params:  raw,
		}
		if err := conn.WriteJSON(req); err != nil {
			t.Fatalf("WriteJSON %d failed: %v", i, err)
		}
	}

	// Read all responses (order may vary due to concurrent server handling)
	responses := make([]JSONRPCResponse, n)
	for i := 0; i < n; i++ {
		if err := conn.ReadJSON(&responses[i]); err != nil {
			t.Fatalf("ReadJSON failed at %d: %v", i, err)
		}
	}

	for i, resp := range responses {
		if resp.JSONRPC != "2.0" {
			t.Errorf("response %d: jsonrpc = %q, want '2.0'", i, resp.JSONRPC)
		}
	}
}

func TestWebSocket_MultipleConnections(t *testing.T) {
	s, ts := newWSTestServer(t, Options{})
	s.tools["svc.Echo"] = &Tool{
		Name:        "svc.Echo",
		Description: "Echo",
		InputSchema: map[string]interface{}{"type": "object"},
		Service:     "svc",
		Endpoint:    "Echo",
	}

	// Connect multiple clients simultaneously
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			conn := wsDialer(t, ts.URL+"/mcp/ws", nil)
			resp := sendJSONRPC(t, conn, "tools/list", idx+1, nil)
			if resp.Error != nil {
				t.Errorf("client %d: unexpected error: %v", idx, resp.Error)
			}
		}(i)
	}
	wg.Wait()
}

func TestWebSocket_InvalidJSON(t *testing.T) {
	_, ts := newWSTestServer(t, Options{})

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/mcp/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Send invalid JSON
	conn.WriteMessage(websocket.TextMessage, []byte("not json"))

	var resp JSONRPCResponse
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != ParseError {
		t.Errorf("expected ParseError, got %+v", resp.Error)
	}
}

func TestWebSocket_InvalidJSONRPCVersion(t *testing.T) {
	_, ts := newWSTestServer(t, Options{})
	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)

	req := map[string]interface{}{
		"jsonrpc": "1.0",
		"id":      1,
		"method":  "initialize",
	}
	conn.WriteJSON(req)

	var resp JSONRPCResponse
	conn.ReadJSON(&resp)
	if resp.Error == nil || resp.Error.Code != InvalidRequest {
		t.Errorf("expected InvalidRequest, got %+v", resp.Error)
	}
}

func TestWebSocket_ConnectionPersistence(t *testing.T) {
	_, ts := newWSTestServer(t, Options{})
	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)

	// Send multiple sequential requests on the same connection
	for i := 0; i < 3; i++ {
		resp := sendJSONRPC(t, conn, "initialize", i+1, nil)
		if resp.Error != nil {
			t.Errorf("request %d: unexpected error: %v", i, resp.Error)
		}
	}

	// Connection should still be alive after a short delay
	time.Sleep(50 * time.Millisecond)
	resp := sendJSONRPC(t, conn, "initialize", 99, nil)
	if resp.Error != nil {
		t.Errorf("request after delay: unexpected error: %v", resp.Error)
	}
}
