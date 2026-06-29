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
	res := NewManualResolver().Add(
		Tool{Name: "echo", Description: "echoes text", InputSchema: map[string]interface{}{
			"type": "object", "properties": map[string]interface{}{"text": map[string]interface{}{"type": "string"}},
		}},
		func(_ context.Context, args map[string]interface{}) (string, error) {
			s, _ := args["text"].(string)
			return "you said: " + s, nil
		},
	)
	ts := httptest.NewServer(NewHandler(res))
	defer ts.Close()

	rpc := func(body string) map[string]interface{} {
		resp, err := http.Post(ts.URL, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("post: %v", err)
		}
		defer resp.Body.Close()
		var out map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&out)
		return out
	}

	// initialize
	if out := rpc(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`); out["result"] == nil {
		t.Fatalf("initialize: %v", out)
	}
	// tools/list
	out := rpc(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	tools := out["result"].(map[string]interface{})["tools"].([]interface{})
	if len(tools) != 1 || tools[0].(map[string]interface{})["name"] != "echo" {
		t.Fatalf("tools/list: %v", out)
	}
	// tools/call
	out = rpc(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hi"}}}`)
	content := out["result"].(map[string]interface{})["content"].([]interface{})
	got := content[0].(map[string]interface{})["text"].(string)
	if got != "you said: hi" {
		t.Fatalf("tools/call text = %q", got)
	}
	// unknown tool -> error
	out = rpc(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nope","arguments":{}}}`)
	if out["error"] == nil {
		t.Fatalf("expected error for unknown tool, got %v", out)
	}
}
