package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManualResolverHandler(t *testing.T) {
	res := NewManualResolver().
		Add(Tool{Name: "echo", Description: "echoes text"},
			func(_ context.Context, args map[string]interface{}) (*CallResult, error) {
				s, _ := args["text"].(string)
				return &CallResult{Text: "you said: " + s}, nil
			}).
		Add(Tool{Name: "boom", Description: "errors"},
			func(_ context.Context, _ map[string]interface{}) (*CallResult, error) {
				return &CallResult{Text: "kaboom", IsError: true}, nil
			}).
		Add(Tool{Name: "blocked", Description: "coded error"},
			func(_ context.Context, _ map[string]interface{}) (*CallResult, error) {
				return nil, &RPCError{Code: -32000, Message: "insufficient credits"}
			})

	ts := httptest.NewServer(NewHandler(res))
	defer ts.Close()
	rpc := func(body string) (int, map[string]interface{}) {
		resp, err := http.Post(ts.URL, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("post rpc: %v", err)
		}
		defer resp.Body.Close()
		var out map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}

	if _, out := rpc(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`); len(out["result"].(map[string]interface{})["tools"].([]interface{})) != 3 {
		t.Fatalf("tools/list: %v", out)
	}
	// tool result
	_, out := rpc(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hi"}}}`)
	if out["result"].(map[string]interface{})["content"].([]interface{})[0].(map[string]interface{})["text"] != "you said: hi" {
		t.Fatalf("echo: %v", out)
	}
	// tool-level error -> isError result, NOT protocol error
	_, out = rpc(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"boom","arguments":{}}}`)
	if out["error"] != nil || out["result"].(map[string]interface{})["isError"] != true {
		t.Fatalf("boom should be isError result: %v", out)
	}
	// coded protocol error -> JSON-RPC error with the code
	_, out = rpc(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"blocked","arguments":{}}}`)
	if out["error"] == nil || int(out["error"].(map[string]interface{})["code"].(float64)) != -32000 {
		t.Fatalf("blocked should be -32000: %v", out)
	}
	// notification -> 204, no body
	code, _ := rpc(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if code != http.StatusNoContent {
		t.Fatalf("notification status = %d, want 204", code)
	}
}
