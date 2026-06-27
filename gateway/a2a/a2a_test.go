package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
	"go-micro.dev/v6/server"
)

// echoAgent is a stub that implements the Agent proto handler — enough to
// exercise the gateway's task→Agent.Chat translation without pulling in
// the agent package (which would import this one, a test-only cycle).
type echoAgent struct{}

func (echoAgent) Chat(_ context.Context, req *pb.ChatRequest, rsp *pb.ChatResponse) error {
	rsp.Reply = "pong"
	rsp.Agent = "echo"
	return nil
}

func waitFor(reg registry.Registry, name string) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if svcs, err := reg.GetService(name); err == nil && len(svcs) > 0 && len(svcs[0].Nodes) > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func newGatewayWithAgent(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	reg := registry.NewMemoryRegistry()
	cl := client.NewClient(client.Registry(reg), client.Selector(selector.NewSelector(selector.Registry(reg))))

	srv := server.NewServer(
		server.Name("echo"),
		server.Address("127.0.0.1:0"),
		server.Registry(reg),
		server.Metadata(map[string]string{"type": "agent", "services": ""}),
	)
	if err := pb.RegisterAgentHandler(srv, echoAgent{}); err != nil {
		t.Fatalf("register agent handler: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	waitFor(reg, "echo")

	g := New(Options{Registry: reg, Client: cl, BaseURL: "http://gw"})
	ts := httptest.NewServer(g.Handler())
	return ts, func() { ts.Close(); srv.Stop() }
}

func TestAgentCardFromRegistry(t *testing.T) {
	ts, cleanup := newGatewayWithAgent(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/agents/echo/.well-known/agent.json")
	if err != nil {
		t.Fatalf("get card: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("card status = %d", resp.StatusCode)
	}
	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		t.Fatalf("decode card: %v", err)
	}
	if card.Name != "echo" {
		t.Errorf("card name = %q, want echo", card.Name)
	}
	if card.URL != "http://gw/agents/echo" {
		t.Errorf("card url = %q", card.URL)
	}
	if card.ProtocolVersion == "" || len(card.Skills) == 0 {
		t.Errorf("card missing protocolVersion or skills: %+v", card)
	}
}

func TestMessageSendAndGet(t *testing.T) {
	ts, cleanup := newGatewayWithAgent(t)
	defer cleanup()

	task := rpcTask(t, ts.URL+"/agents/echo", `{
		"jsonrpc":"2.0","id":1,"method":"message/send",
		"params":{"message":{"role":"user","kind":"message","messageId":"m1",
			"parts":[{"kind":"text","text":"ping"}]}}}`)
	if task.Status.State != stateCompleted {
		t.Fatalf("task state = %q, want completed", task.Status.State)
	}
	if len(task.Artifacts) != 1 || textOf(task.Artifacts[0].Parts) != "pong" {
		t.Fatalf("artifact = %+v, want text 'pong'", task.Artifacts)
	}
	if len(task.History) != 2 || task.History[1].Role != "agent" || textOf(task.History[1].Parts) != "pong" {
		t.Fatalf("history = %+v, want user turn followed by agent reply", task.History)
	}
	if task.History[1].TaskID != task.ID || task.History[1].ContextID != task.ContextID {
		t.Fatalf("agent history linkage = task %q/%q context %q/%q", task.History[1].TaskID, task.ID, task.History[1].ContextID, task.ContextID)
	}

	got := rpcTask(t, ts.URL+"/agents/echo", `{
		"jsonrpc":"2.0","id":2,"method":"tasks/get","params":{"id":"`+task.ID+`"}}`)
	if got.ID != task.ID || got.Status.State != stateCompleted {
		t.Errorf("tasks/get returned %+v", got)
	}
}

func TestMessageSendUsesRequestContext(t *testing.T) {
	d := newDispatcher()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{
		"jsonrpc":"2.0","id":1,"method":"message/send",
		"params":{"message":{"role":"user","kind":"message","messageId":"m1",
			"parts":[{"kind":"text","text":"ping"}]}}}`))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	d.serve(rr, req, func(ctx context.Context, text string) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return "unexpected success", nil
	})

	var resp struct {
		Result Task      `json:"result"`
		Error  *rpcError `json:"error"`
	}
	if err := json.NewDecoder(rr.Result().Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("rpc error: %+v", resp.Error)
	}
	if resp.Result.Status.State != stateFailed {
		t.Fatalf("task state = %q, want failed", resp.Result.Status.State)
	}
	if len(resp.Result.Artifacts) != 1 || textOf(resp.Result.Artifacts[0].Parts) != "error: context canceled" {
		t.Fatalf("artifact = %+v, want context cancellation", resp.Result.Artifacts)
	}
	if len(resp.Result.History) != 2 || resp.Result.History[1].Role != "agent" || textOf(resp.Result.History[1].Parts) != "error: context canceled" {
		t.Fatalf("history = %+v, want failed agent reply recorded", resp.Result.History)
	}
}

func TestMessageStream(t *testing.T) {
	ts, cleanup := newGatewayWithAgent(t)
	defer cleanup()

	body := `{"jsonrpc":"2.0","id":1,"method":"message/stream","params":{"message":{"role":"user","parts":[{"kind":"text","text":"ping"}],"kind":"message"}}}`
	resp, err := http.Post(ts.URL+"/agents/echo", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}

	var line string
	if _, err := fmt.Fscan(resp.Body, &line); err != nil {
		t.Fatalf("read event prefix: %v", err)
	}
	if line != "data:" {
		t.Fatalf("event prefix = %q, want data:", line)
	}
	var out struct {
		Result Task      `json:"result"`
		Error  *rpcError `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if out.Error != nil {
		t.Fatalf("rpc error: %+v", out.Error)
	}
	if out.Result.Status.State != stateCompleted || len(out.Result.Artifacts) != 1 || textOf(out.Result.Artifacts[0].Parts) != "pong" {
		t.Fatalf("streamed task = %+v", out.Result)
	}
}

func TestUnknownMethod(t *testing.T) {
	ts, cleanup := newGatewayWithAgent(t)
	defer cleanup()

	var resp struct {
		Error *rpcError `json:"error"`
	}
	rpc(t, ts.URL+"/agents/echo", `{"jsonrpc":"2.0","id":1,"method":"tasks/resubscribe","params":{}}`, &resp)
	if resp.Error == nil || resp.Error.Code != errMethodNotFound {
		t.Errorf("expected method-not-found for resubscribe, got %+v", resp.Error)
	}
}

func TestListAgents(t *testing.T) {
	ts, cleanup := newGatewayWithAgent(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/agents")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer resp.Body.Close()
	var out struct {
		Agents []AgentCard `json:"agents"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Agents) != 1 || out.Agents[0].Name != "echo" {
		t.Errorf("agents list = %+v", out.Agents)
	}
}

func rpc(t *testing.T, url, body string, v any) {
	t.Helper()
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func rpcTask(t *testing.T, url, body string) Task {
	t.Helper()
	var resp struct {
		Result Task      `json:"result"`
		Error  *rpcError `json:"error"`
	}
	rpc(t, url, body, &resp)
	if resp.Error != nil {
		t.Fatalf("rpc error: %+v", resp.Error)
	}
	return resp.Result
}
