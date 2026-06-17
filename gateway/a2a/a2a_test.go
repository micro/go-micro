package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-micro.dev/v5/agent"
	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/selector"
	"go-micro.dev/v5/store"
)

// mockModel is a fixed-reply LLM so the test needs no API key.
type mockModel struct{ opts ai.Options }

func newMock(opts ...ai.Option) ai.Model { m := &mockModel{}; _ = m.Init(opts...); return m }
func (m *mockModel) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *mockModel) Options() ai.Options { return m.opts }
func (m *mockModel) String() string      { return "mock" }
func (m *mockModel) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) {
	return nil, fmt.Errorf("no stream")
}
func (m *mockModel) Generate(context.Context, *ai.Request, ...ai.GenerateOption) (*ai.Response, error) {
	return &ai.Response{Answer: "pong"}, nil
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
	ai.Register("mock", newMock)

	a := agent.New(
		agent.Name("echo"),
		agent.Provider("mock"),
		agent.WithRegistry(reg),
		agent.WithClient(cl),
		agent.WithStore(store.NewMemoryStore()),
	)
	go a.Run()
	waitFor(reg, "echo")

	g := New(Options{Registry: reg, Client: cl, BaseURL: "http://gw"})
	ts := httptest.NewServer(g.Handler())
	return ts, func() { ts.Close(); a.Stop() }
}

func TestAgentCardFromRegistry(t *testing.T) {
	ts, cleanup := newGatewayWithAgent(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/agents/echo/.well-known/agent.json")
	if err != nil {
		t.Fatalf("get card: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
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

	// message/send -> completed task with the agent's reply.
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

	// tasks/get -> the same task, by id.
	got := rpcTask(t, ts.URL+"/agents/echo", fmt.Sprintf(`{
		"jsonrpc":"2.0","id":2,"method":"tasks/get","params":{"id":%q}}`, task.ID))
	if got.ID != task.ID || got.Status.State != stateCompleted {
		t.Errorf("tasks/get returned %+v", got)
	}
}

func TestUnknownMethod(t *testing.T) {
	ts, cleanup := newGatewayWithAgent(t)
	defer cleanup()

	var resp struct {
		Error *rpcError `json:"error"`
	}
	rpc(t, ts.URL+"/agents/echo", `{"jsonrpc":"2.0","id":1,"method":"message/stream","params":{}}`, &resp)
	if resp.Error == nil || resp.Error.Code != errMethodNotFound {
		t.Errorf("expected method-not-found for streaming, got %+v", resp.Error)
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

// rpc posts a JSON-RPC request and decodes the response into v.
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

// rpcTask posts a JSON-RPC request and returns the Task result.
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
