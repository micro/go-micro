package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newStreamServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	s := newTestServer(Options{})
	s.tools["echo.Echo"] = &Tool{
		Name:        "echo.Echo",
		Description: "Echo input back",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		Handler: func(input map[string]interface{}) (interface{}, error) {
			return input, nil
		},
	}
	ts := httptest.NewServer(s.handler())
	t.Cleanup(ts.Close)
	return s, ts
}

// rpcRequest posts a JSON-RPC body to /mcp and returns the raw response.
func rpcRequest(t *testing.T, ts *httptest.Server, sessionID string, body string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set(mcpSessionHeader, sessionID)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func decodeRPC(t *testing.T, data []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("not a JSON object: %s", data)
	}
	return m
}

func TestStreamable_Handshake(t *testing.T) {
	_, ts := newStreamServer(t)

	resp, data := rpcRequest(t, ts, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body: %s", resp.StatusCode, data)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type = %q", ct)
	}
	sid := resp.Header.Get(mcpSessionHeader)
	if sid == "" {
		t.Fatal("initialize response missing Mcp-Session-Id")
	}

	m := decodeRPC(t, data)
	if m["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v", m["jsonrpc"])
	}
	result, _ := m["result"].(map[string]interface{})
	if result["protocolVersion"] != "2025-06-18" {
		t.Errorf("protocolVersion = %v", result["protocolVersion"])
	}
	if m["error"] != nil {
		t.Errorf("unexpected error: %v", m["error"])
	}

	// notifications/initialized → 202, no body.
	resp, data = rpcRequest(t, ts, sid, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("notification status = %d, want 202", resp.StatusCode)
	}
	if len(data) != 0 {
		t.Errorf("notification body = %s, want empty", data)
	}
}

func TestStreamable_ToolsListAndCall(t *testing.T) {
	_, ts := newStreamServer(t)

	// initialize to get a session
	_, data := rpcRequest(t, ts, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{}}}`)
	sid := mustSessionID(t, ts)
	_ = data

	// tools/list
	_, data = rpcRequest(t, ts, sid, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	m := decodeRPC(t, data)
	result, _ := m["result"].(map[string]interface{})
	tools, _ := result["tools"].([]interface{})
	if len(tools) == 0 {
		t.Fatal("expected tools in tools/list result")
	}
	found := false
	for _, ti := range tools {
		if tm, _ := ti.(map[string]interface{}); tm["name"] == "echo.Echo" {
			found = true
		}
	}
	if !found {
		t.Errorf("echo.Echo not listed: %v", tools)
	}

	// tools/call
	_, data = rpcRequest(t, ts, sid, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo.Echo","arguments":{"msg":"hi"}}}`)
	m = decodeRPC(t, data)
	result, _ = m["result"].(map[string]interface{})
	content, _ := result["content"].([]interface{})
	if len(content) == 0 {
		t.Fatal("expected content in tools/call result")
	}
	text := content[0].(map[string]interface{})["text"]
	if text != `{"msg":"hi"}` {
		t.Errorf("content text = %v", text)
	}
}

func TestStreamable_UnknownTool(t *testing.T) {
	_, ts := newStreamServer(t)
	_, data := rpcRequest(t, ts, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	sid := mustSessionID(t, ts)
	_ = data

	_, data = rpcRequest(t, ts, sid, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nope.Do","arguments":{}}}`)
	m := decodeRPC(t, data)
	errObj, _ := m["error"].(map[string]interface{})
	if errObj == nil {
		t.Fatalf("expected error, got: %s", data)
	}
	if errObj["code"] != float64(-32602) {
		t.Errorf("error code = %v, want -32602", errObj["code"])
	}
}

func TestStreamable_MethodNotFound(t *testing.T) {
	_, ts := newStreamServer(t)
	_, data := rpcRequest(t, ts, "", `{"jsonrpc":"2.0","id":5,"method":"prompts/list"}`)
	m := decodeRPC(t, data)
	errObj, _ := m["error"].(map[string]interface{})
	if errObj == nil || errObj["code"] != float64(MethodNotFound) {
		t.Errorf("expected method not found, got: %s", data)
	}
}

func TestStreamable_Batch(t *testing.T) {
	_, ts := newStreamServer(t)
	_, data := rpcRequest(t, ts, "", `[{"jsonrpc":"2.0","id":6,"method":"ping"},{"jsonrpc":"2.0","id":7,"method":"ping"}]`)
	var batch []map[string]interface{}
	if err := json.Unmarshal(data, &batch); err != nil {
		t.Fatalf("expected batch array, got: %s", data)
	}
	if len(batch) != 2 {
		t.Fatalf("batch length = %d", len(batch))
	}
	for _, m := range batch {
		if m["error"] != nil || m["result"] == nil {
			t.Errorf("bad batch element: %v", m)
		}
	}
}

func TestStreamable_SessionLifecycle(t *testing.T) {
	s, ts := newStreamServer(t)

	_, data := rpcRequest(t, ts, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	sid := mustSessionID(t, ts)
	_ = data

	// Open the GET SSE stream for the session.
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set(mcpSessionHeader, sid)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("content-type = %q", ct)
	}

	// Push a server→client message and read it off the stream.
	sess := s.session(sid)
	if sess == nil {
		t.Fatal("session not registered")
	}
	s.emit(sess, []byte(`{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`))

	reader := bufio.NewReader(resp.Body)
	var got []string
	deadline := time.Now().Add(5 * time.Second)
	for len(got) < 3 && time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		got = append(got, strings.TrimRight(line, "\r\n"))
	}
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "event: message") || !strings.Contains(joined, "notifications/tools/list_changed") {
		t.Errorf("SSE stream did not deliver the event. got:\n%s", joined)
	}

	// DELETE terminates the session.
	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/mcp", nil)
	delReq.Header.Set(mcpSessionHeader, sid)
	delResp, err := ts.Client().Do(delReq)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE status = %d, want 204", delResp.StatusCode)
	}
	if s.session(sid) != nil {
		t.Error("session still registered after DELETE")
	}
}

func TestStreamable_GETWithoutSession(t *testing.T) {
	_, ts := newStreamServer(t)
	resp, err := ts.Client().Get(ts.URL + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET without session status = %d, want 405", resp.StatusCode)
	}
}

func TestStreamable_ParseError(t *testing.T) {
	_, ts := newStreamServer(t)
	resp, data := rpcRequest(t, ts, "", `not json`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	m := decodeRPC(t, data)
	errObj, _ := m["error"].(map[string]interface{})
	if errObj == nil || errObj["code"] != float64(ParseError) {
		t.Errorf("expected parse error, got: %s", data)
	}
}

func mustSessionID(t *testing.T, ts *httptest.Server) string {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/mcp", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":99,"method":"initialize","params":{}}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	sid := resp.Header.Get(mcpSessionHeader)
	if sid == "" {
		t.Fatal("no session id in initialize response")
	}
	return sid
}
