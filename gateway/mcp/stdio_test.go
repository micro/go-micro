package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go-micro.dev/v6/client"
)

// fakeCallClient overrides Call to return canned data or an error; NewRequest
// and the rest are promoted from the embedded real client.
type fakeCallClient struct {
	client.Client
	data []byte
	err  error
}

func (f *fakeCallClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	if f.err != nil {
		return f.err
	}
	if r, ok := rsp.(*struct{ Data []byte }); ok {
		r.Data = f.data
	}
	return nil
}

// isToolError reports whether an MCP tools/call result carries isError:true.
func isToolError(result interface{}) bool {
	m, ok := result.(map[string]interface{})
	if !ok {
		return false
	}
	b, _ := m["isError"].(bool)
	return b
}

// toolResultText extracts the first text content of an MCP tools/call result.
func toolResultText(t *testing.T, result interface{}) string {
	t.Helper()
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %#v", result)
	}
	content, ok := m["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatalf("result has no content: %#v", result)
	}
	first, _ := content[0].(map[string]interface{})
	text, _ := first["text"].(string)
	return text
}

// driveStdio sends one JSON-RPC request through a StdioTransport and returns the
// decoded response, capturing the transport's stdout into a buffer.
func driveStdio(t *testing.T, s *Server, method string, id interface{}, params interface{}) JSONRPCResponse {
	t.Helper()
	tr := NewStdioTransport(s)
	var out bytes.Buffer
	tr.writer = bufio.NewWriter(&out)
	raw, _ := json.Marshal(params)
	tr.handleRequest(&JSONRPCRequest{JSONRPC: "2.0", ID: id, Method: method, Params: raw})
	var resp JSONRPCResponse
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("decode stdio response: %v (raw=%q)", err, out.String())
	}
	return resp
}

// The stdio transport is the path an external MCP host (Claude Desktop) uses.
// It must return tool output as JSON text, not fmt.Sprintf("%v", ...) which
// yields Go map-syntax and is unparseable by a real client.
func TestStdio_ToolsCall_ReturnsJSONNotGoSyntax(t *testing.T) {
	s := newTestServer(Options{})
	s.opts.Client = &fakeCallClient{Client: client.DefaultClient, data: []byte(`{"id":1,"name":"bob"}`)}
	s.tools["svc.Echo"] = &Tool{Name: "svc.Echo", Service: "svc", Endpoint: "Echo"}

	resp := driveStdio(t, s, "tools/call", 1, map[string]interface{}{
		"name":      "svc.Echo",
		"arguments": map[string]interface{}{"msg": "hi"},
	})
	if resp.Error != nil {
		t.Fatalf("unexpected protocol error: %+v", resp.Error)
	}
	text := toolResultText(t, resp.Result)
	// The bug returned Go map-syntax ("map[id:1 name:bob]"), which fails to parse.
	var got map[string]interface{}
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("tool result text is not JSON (the %%v bug): %q", text)
	}
	if got["name"] != "bob" {
		t.Errorf("result = %v, want name=bob", got)
	}
}

// A tool-execution failure must be an MCP isError result, not a JSON-RPC
// protocol error, so the agent can read the failure.
func TestStdio_ToolsCall_FailureIsIsErrorResult(t *testing.T) {
	s := newTestServer(Options{})
	s.opts.Client = &fakeCallClient{Client: client.DefaultClient, err: errors.New("backend down")}
	s.tools["svc.Echo"] = &Tool{Name: "svc.Echo", Service: "svc", Endpoint: "Echo"}

	resp := driveStdio(t, s, "tools/call", 1, map[string]interface{}{
		"name":      "svc.Echo",
		"arguments": map[string]interface{}{},
	})
	if resp.Error != nil {
		t.Fatalf("tool failure returned a protocol error, want isError result: %+v", resp.Error)
	}
	if !isToolError(resp.Result) {
		t.Fatalf("expected isError result, got %+v", resp.Result)
	}
	if text := toolResultText(t, resp.Result); text == "" {
		t.Error("isError result should carry the error text")
	}
}
