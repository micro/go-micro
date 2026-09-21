package mcp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"go-micro.dev/v6/auth"
	"go-micro.dev/v6/wrapper/x402"
)

func TestBrowserOriginsAcrossTransports(t *testing.T) {
	s := newTestServer(Options{})
	for name, handler := range map[string]http.Handler{
		"streamable":  http.HandlerFunc(s.handleStreamableHTTP),
		"legacy-call": http.HandlerFunc(s.handleCallTool),
		"legacy-list": http.HandlerFunc(s.handleListTools),
		"websocket":   NewWebSocketTransport(s),
		"embedded":    NewHandler(nil),
	} {
		t.Run(name, func(t *testing.T) {
			for _, origin := range []string{"https://untrusted.example", "null", "http://localhost:1234/path", ""} {
				req := httptest.NewRequest(http.MethodPost, "http://localhost:1234/mcp", strings.NewReader(`{}`))
				req.Header["Origin"] = []string{origin}
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				if rr.Code != http.StatusForbidden {
					t.Fatalf("origin %q: status %d", origin, rr.Code)
				}
			}
		})
	}
	s.opts.AllowedOrigins = []string{"https://trusted.example"}
	for _, origin := range []string{"https://trusted.example", "http://localhost:1234", ""} {
		req := httptest.NewRequest(http.MethodOptions, "http://localhost:1234/mcp", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		rr := httptest.NewRecorder()
		s.handleStreamableHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("allowed origin %q: %d", origin, rr.Code)
		}
		if rr.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Fatal("unexpected CORS origin")
		}
	}
}

func TestWebSocketHonorsAuthFuncBeforeUpgrade(t *testing.T) {
	s := newTestServer(Options{AuthFunc: func(*http.Request) error { return errors.New("denied") }})
	rr := httptest.NewRecorder()
	NewWebSocketTransport(s).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/mcp/ws", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want auth rejection before upgrade", rr.Code)
	}
}

type revocableTestAuth struct {
	auth.Auth
	revoked atomic.Bool
}

func (a *revocableTestAuth) Inspect(token string) (*auth.Account, error) {
	if a.revoked.Load() {
		return nil, errors.New("revoked")
	}
	return a.Auth.Inspect(token)
}
func TestWebSocketRechecksConnectionCredential(t *testing.T) {
	provider := &revocableTestAuth{Auth: &mockAuth{accounts: map[string]*auth.Account{"credential": {ID: "caller", Scopes: []string{"*"}}}}}
	s, ts := newWSTestServer(t, Options{Auth: provider})
	s.tools["svc.Echo"] = &Tool{Name: "svc.Echo", Service: "svc", Endpoint: "Echo"}
	headers := http.Header{"Authorization": []string{"Bearer credential"}}
	conn := wsDialer(t, ts.URL+"/mcp/ws", headers)
	first := sendJSONRPC(t, conn, "tools/call", 1, map[string]any{"name": "svc.Echo"})
	if first.Error != nil && first.Error.Message == "Unauthorized" {
		t.Fatal("valid credential rejected")
	}
	provider.revoked.Store(true)
	second := sendJSONRPC(t, conn, "tools/call", 2, map[string]any{"name": "svc.Echo"})
	if second.Error == nil || second.Error.Message != "Unauthorized" {
		t.Fatalf("revoked credential accepted: %+v", second)
	}
}

func TestStreamableHonorsAuthFunc(t *testing.T) {
	s := newTestServer(Options{AuthFunc: func(*http.Request) error { return errors.New("denied") }})
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodDelete} {
		rr := httptest.NewRecorder()
		s.handleStreamableHTTP(rr, httptest.NewRequest(method, "/mcp", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s: got %d", method, rr.Code)
		}
	}
}
func TestWebSocketDoesNotBypassPayment(t *testing.T) {
	s, ts := newWSTestServer(t, Options{Payment: &x402.Config{Amount: "10"}})
	s.tools["svc.Paid"] = &Tool{Name: "svc.Paid", Service: "svc", Endpoint: "Paid"}
	conn := wsDialer(t, ts.URL+"/mcp/ws", nil)
	response := sendJSONRPC(t, conn, "tools/call", 1, map[string]any{"name": "svc.Paid"})
	if response.Error == nil || response.Error.Message != "Payment required" {
		t.Fatalf("paid call not rejected: %+v", response)
	}
}
