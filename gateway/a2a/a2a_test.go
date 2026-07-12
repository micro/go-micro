package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/ai"
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
		server.Metadata(map[string]string{"type": "agent", "services": "task,project"}),
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
	if card.ProtocolVersion == "" {
		t.Errorf("card missing protocolVersion: %+v", card)
	}
	if !card.Capabilities.TaskResubscribe || !card.Capabilities.InputRequired {
		t.Errorf("card capabilities = %+v, want task resubscribe and input-required advertised", card.Capabilities)
	}
	if got := skillIDs(card.Skills); strings.Join(got, ",") != "task,project" {
		t.Errorf("skill IDs = %v, want [task project]", got)
	}
}

// A2A 0.3.0 discovery is /.well-known/agent-card.json. The card must be
// reachable there (canonical) as well as at the legacy agent.json alias, both
// per-agent and at the single-agent top level.
func TestAgentCardCanonicalWellKnownPath(t *testing.T) {
	ts, cleanup := newGatewayWithAgent(t)
	defer cleanup()

	for _, path := range []string{
		"/agents/echo/.well-known/agent-card.json",
		"/agents/echo/.well-known/agent.json",
		"/agents/echo/skills/task/.well-known/agent-card.json",
	} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			t.Fatalf("%s status = %d, want 200", path, resp.StatusCode)
		}
		var card AgentCard
		if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
			resp.Body.Close()
			t.Fatalf("%s decode card: %v", path, err)
		}
		resp.Body.Close()
		if card.Name != "echo" {
			t.Errorf("%s card name = %q, want echo", path, card.Name)
		}
	}
}

func TestSkillEndpointServesFocusedCardAndRoutesRPC(t *testing.T) {
	ts, cleanup := newGatewayWithAgent(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/agents/echo/skills/task/.well-known/agent.json")
	if err != nil {
		t.Fatalf("get skill card: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("skill card status = %d", resp.StatusCode)
	}
	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		t.Fatalf("decode skill card: %v", err)
	}
	if card.URL != "http://gw/agents/echo/skills/task" || len(card.Skills) != 1 || card.Skills[0].ID != "task" {
		t.Fatalf("skill card = %+v, want task-only card at skill URL", card)
	}

	task := rpcTask(t, ts.URL+"/agents/echo/skills/task", `{
		"jsonrpc":"2.0","id":1,"method":"message/send",
		"params":{"message":{"role":"user","kind":"message","messageId":"m1",
			"parts":[{"kind":"text","text":"ping"}]}}}`)
	if task.Status.State != stateCompleted || textOf(task.Artifacts[0].Parts) != "pong" {
		t.Fatalf("skill task = %+v, want completed pong", task)
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

func TestMessageSendContinuesExistingTask(t *testing.T) {
	d := newDispatcher()
	first := rpcTaskFromBody(t, d, `{
		"jsonrpc":"2.0","id":1,"method":"message/send",
		"params":{"message":{"role":"user","kind":"message","messageId":"m1",
			"parts":[{"kind":"text","text":"first"}]}}}`, func(_ context.Context, text string) (string, error) {
		return "reply to " + text, nil
	})

	secondBody := fmt.Sprintf(`{
		"jsonrpc":"2.0","id":2,"method":"message/send",
		"params":{"message":{"role":"user","kind":"message","messageId":"m2","taskId":"%s","contextId":"%s",
			"parts":[{"kind":"text","text":"second"}]}}}`, first.ID, first.ContextID)
	second := rpcTaskFromBody(t, d, secondBody, func(_ context.Context, text string) (string, error) {
		return "reply to " + text, nil
	})

	if second.ID != first.ID || second.ContextID != first.ContextID {
		t.Fatalf("continued task identity = %s/%s, want %s/%s", second.ID, second.ContextID, first.ID, first.ContextID)
	}
	if len(second.History) != 4 {
		t.Fatalf("continued history len = %d, want 4: %+v", len(second.History), second.History)
	}
	if textOf(second.History[0].Parts) != "first" || textOf(second.History[2].Parts) != "second" {
		t.Fatalf("continued history did not preserve turns: %+v", second.History)
	}

	got := rpcTaskFromDispatcher(t, d, first.ID)
	if got.ID != first.ID || len(got.History) != 4 {
		t.Fatalf("stored continued task = %+v", got)
	}
}

func TestPushNotificationConfigDeliversTaskUpdates(t *testing.T) {
	d := newDispatcher()
	updates := make(chan Task, 2)
	push := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("authorization = %q, want bearer token", got)
		}
		var task Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			t.Errorf("decode push task: %v", err)
			return
		}
		updates <- task
		w.WriteHeader(http.StatusAccepted)
	}))
	defer push.Close()

	task := rpcTaskFromBody(t, d, `{
		"jsonrpc":"2.0","id":1,"method":"message/send",
		"params":{"message":{"role":"user","kind":"message","messageId":"m1",
			"parts":[{"kind":"text","text":"ping"}]}}}`, func(_ context.Context, text string) (string, error) {
		return "pong", nil
	})

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tasks/pushNotificationConfig/set","params":{"id":"%s","pushNotificationConfig":{"url":"%s","token":"secret"}}}`, task.ID, push.URL)
	var setResp struct {
		Result struct {
			ID                     string                 `json:"id"`
			PushNotificationConfig PushNotificationConfig `json:"pushNotificationConfig"`
		} `json:"result"`
		Error *rpcError `json:"error"`
	}
	rpcDispatcher(t, d, body, nil, &setResp)
	if setResp.Error != nil {
		t.Fatalf("set push config error: %+v", setResp.Error)
	}
	if setResp.Result.ID != task.ID || setResp.Result.PushNotificationConfig.URL != push.URL {
		t.Fatalf("set push config result = %+v", setResp.Result)
	}

	select {
	case got := <-updates:
		if got.ID != task.ID || got.Status.State != stateCompleted || textOf(got.Artifacts[0].Parts) != "pong" {
			t.Fatalf("push update = %+v, want completed task", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for push update")
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

type sliceStream struct {
	chunks []string
	err    error
}

func (s *sliceStream) Recv() (*ai.Response, error) {
	if len(s.chunks) == 0 {
		if s.err != nil {
			err := s.err
			s.err = nil
			return nil, err
		}
		return nil, io.EOF
	}
	next := s.chunks[0]
	s.chunks = s.chunks[1:]
	return &ai.Response{Reply: next}, nil
}

func (s *sliceStream) Close() error { return nil }

// streamEvent is one decoded SSE JSON-RPC event from a message/stream response.
// A2A streams carry heterogeneous results (Task, status-update, artifact-update)
// discriminated by `kind`, so we keep the raw result and decode on demand.
type streamEvent struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

func (e streamEvent) kind() string {
	var k struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal(e.Result, &k)
	return k.Kind
}

func (e streamEvent) task(t *testing.T) Task {
	t.Helper()
	var task Task
	if err := json.Unmarshal(e.Result, &task); err != nil {
		t.Fatalf("decode task event: %v", err)
	}
	return task
}

func (e streamEvent) status(t *testing.T) TaskStatusUpdateEvent {
	t.Helper()
	var s TaskStatusUpdateEvent
	if err := json.Unmarshal(e.Result, &s); err != nil {
		t.Fatalf("decode status-update event: %v", err)
	}
	return s
}

func (e streamEvent) artifactUpdate(t *testing.T) TaskArtifactUpdateEvent {
	t.Helper()
	var a TaskArtifactUpdateEvent
	if err := json.Unmarshal(e.Result, &a); err != nil {
		t.Fatalf("decode artifact-update event: %v", err)
	}
	return a
}

// collectSSE parses the `data:`-framed JSON-RPC events from an SSE body.
func collectSSE(t *testing.T, body string) []streamEvent {
	t.Helper()
	var events []streamEvent
	for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "data:"))
		if line == "" {
			continue
		}
		var e streamEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("decode event %q: %v", line, err)
		}
		events = append(events, e)
	}
	return events
}

func TestMessageStreamChunksStoreFinalTask(t *testing.T) {
	d := newDispatcher()
	body := `{"jsonrpc":"2.0","id":1,"method":"message/stream","params":{"message":{"role":"user","parts":[{"kind":"text","text":"ping"}],"kind":"message"}}}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	d.serveWithStream(rr, req, nil, func(ctx context.Context, text string) (ai.Stream, error) {
		if text != "ping" {
			t.Fatalf("stream text = %q, want ping", text)
		}
		return &sliceStream{chunks: []string{"po", "ng"}}, nil
	})

	if ct := rr.Result().Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}
	events := collectSSE(t, rr.Body.String())
	// Opening Task snapshot + one append artifact-update per chunk + terminal
	// status-update.
	if len(events) != 4 {
		t.Fatalf("events = %d, want 4; body %s", len(events), rr.Body.String())
	}
	for i, e := range events {
		if e.Error != nil {
			t.Fatalf("event %d carried an error field: %+v", i, e.Error)
		}
	}
	if events[0].kind() != "task" {
		t.Fatalf("first event kind = %q, want task", events[0].kind())
	}
	opening := events[0].task(t)
	if opening.Status.State != stateWorking {
		t.Fatalf("opening task state = %q, want working", opening.Status.State)
	}
	taskID := opening.ID

	// The middle events are append artifact-updates carrying the chunk deltas.
	var text strings.Builder
	for _, e := range events[1:3] {
		if e.kind() != "artifact-update" {
			t.Fatalf("event kind = %q, want artifact-update", e.kind())
		}
		au := e.artifactUpdate(t)
		if !au.Append {
			t.Fatalf("artifact-update should be append: %+v", au)
		}
		if au.TaskID != taskID {
			t.Fatalf("artifact-update taskId = %q, want %q", au.TaskID, taskID)
		}
		text.WriteString(textOf(au.Artifact.Parts))
	}
	if text.String() != "pong" {
		t.Fatalf("accumulated artifact text = %q, want pong", text.String())
	}

	// The stream closes with a terminal status-update (final:true).
	last := events[len(events)-1]
	if last.kind() != "status-update" {
		t.Fatalf("last event kind = %q, want status-update", last.kind())
	}
	su := last.status(t)
	if !su.Final || su.Status.State != stateCompleted {
		t.Fatalf("terminal event = %+v, want final completed", su)
	}
	if su.TaskID != taskID {
		t.Fatalf("terminal taskId = %q, want %q", su.TaskID, taskID)
	}

	got := rpcTaskFromDispatcher(t, d, taskID)
	if got.ID != taskID || got.Status.State != stateCompleted || textOf(got.Artifacts[0].Parts) != "pong" {
		t.Fatalf("stored task = %+v, want final completed pong", got)
	}
}

type contextStream struct {
	ctx    context.Context
	closed chan struct{}
}

func (s *contextStream) Recv() (*ai.Response, error) {
	<-s.ctx.Done()
	return nil, s.ctx.Err()
}

func (s *contextStream) Close() error {
	close(s.closed)
	return nil
}

func TestMessageStreamChunksPropagatesCancellationAndClosesStream(t *testing.T) {
	d := newDispatcher()
	ctx, cancel := context.WithCancel(context.Background())
	closed := make(chan struct{})
	body := `{"jsonrpc":"2.0","id":1,"method":"message/stream","params":{"message":{"role":"user","parts":[{"kind":"text","text":"ping"}],"kind":"message"}}}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body)).WithContext(ctx)
	rr := httptest.NewRecorder()
	cancel()

	d.serveWithStream(rr, req, nil, func(ctx context.Context, text string) (ai.Stream, error) {
		if text != "ping" {
			t.Fatalf("stream text = %q, want ping", text)
		}
		return &contextStream{ctx: ctx, closed: closed}, nil
	})

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("stream was not closed")
	}

	events := collectSSE(t, rr.Body.String())
	// Opening Task snapshot, then a terminal failed status-update.
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2; body %s", len(events), rr.Body.String())
	}
	// A streaming failure must be a failed status-update, never `result` and
	// `error` set together in one response.
	for i, e := range events {
		if e.Error != nil {
			t.Fatalf("event %d carried an error field (result+error not allowed): %+v", i, e.Error)
		}
	}
	if events[0].kind() != "task" || events[0].task(t).Status.State != stateWorking {
		t.Fatalf("first event = %s, want working task", string(events[0].Result))
	}
	last := events[1]
	if last.kind() != "status-update" {
		t.Fatalf("last event kind = %q, want status-update", last.kind())
	}
	su := last.status(t)
	if !su.Final || su.Status.State != stateFailed {
		t.Fatalf("terminal event = %+v, want final failed", su)
	}

	got := rpcTaskFromDispatcher(t, d, su.TaskID)
	if got.Status.State != stateFailed || textOf(got.Artifacts[0].Parts) != "error: context canceled" {
		t.Fatalf("stored task = %+v, want failed cancellation", got)
	}
}

func TestMessageStreamChunksFallsBackWhenUnsupported(t *testing.T) {
	d := newDispatcher()
	body := `{"jsonrpc":"2.0","id":1,"method":"message/stream","params":{"message":{"role":"user","parts":[{"kind":"text","text":"ping"}],"kind":"message"}}}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	var streamed bool
	var fallbackText string

	d.serveWithStream(rr, req, func(ctx context.Context, text string) (string, error) {
		fallbackText = text
		return "pong", nil
	}, func(ctx context.Context, text string) (ai.Stream, error) {
		streamed = true
		return nil, fmt.Errorf("%w: test provider", ai.ErrStreamingUnsupported)
	})

	if !streamed {
		t.Fatal("stream invoke was not attempted")
	}
	if fallbackText != "ping" {
		t.Fatalf("fallback text = %q, want ping", fallbackText)
	}
	if ct := rr.Result().Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}
	events := collectSSE(t, rr.Body.String())
	// The non-streaming fallback emits a completed Task snapshot then a terminal
	// status-update.
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2; body %s", len(events), rr.Body.String())
	}
	for i, e := range events {
		if e.Error != nil {
			t.Fatalf("fallback event %d error: %+v", i, e.Error)
		}
	}
	task := events[0].task(t)
	if task.Status.State != stateCompleted || textOf(task.Artifacts[0].Parts) != "pong" {
		t.Fatalf("fallback task = %+v, want completed pong", task)
	}
	su := events[1].status(t)
	if !su.Final || su.Status.State != stateCompleted {
		t.Fatalf("terminal event = %+v, want final completed", su)
	}
}

func TestMessageStreamFallbackDoesNotCompleteWithEmptyText(t *testing.T) {
	d := newDispatcher()
	body := `{"jsonrpc":"2.0","id":1,"method":"message/stream","params":{"message":{"role":"user","parts":[{"kind":"text","text":"ping"}],"kind":"message"}}}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	d.serveWithStream(rr, req, func(context.Context, string) (string, error) {
		return "", nil
	}, func(context.Context, string) (ai.Stream, error) {
		return nil, fmt.Errorf("%w: test provider", ai.ErrStreamingUnsupported)
	})

	events := collectSSE(t, rr.Body.String())
	var task Task
	var foundTask bool
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("fallback event error: %+v", e.Error)
		}
		if e.kind() == "task" {
			task = e.task(t)
			foundTask = true
		}
	}
	if !foundTask {
		t.Fatalf("no task event in stream; body %s", rr.Body.String())
	}
	if task.Status.State != stateFailed {
		t.Fatalf("fallback state = %q, want failed", task.Status.State)
	}
	if got := textOf(task.Artifacts[0].Parts); got == "" {
		t.Fatalf("fallback artifact text is empty: %+v", task.Artifacts)
	}
	if got := textOf(task.History[len(task.History)-1].Parts); got == "" {
		t.Fatalf("fallback history text is empty: %+v", task.History)
	}
	// The stream still ends with a terminal marker.
	last := events[len(events)-1]
	if last.kind() != "status-update" || !last.status(t).Final {
		t.Fatalf("stream must end with a final status-update; got %s", string(last.Result))
	}
}

func TestDecodeAgentChatReplyFallsBackToProviderTextFields(t *testing.T) {
	for name, body := range map[string]string{
		"answer":          `{"answer":"answer text"}`,
		"content":         `{"content":"content text"}`,
		"text":            `{"text":"text field"}`,
		"message_content": `{"message":{"content":"message content"}}`,
		"message_text":    `{"message":{"text":"message text"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := decodeAgentChatReply([]byte(body))
			if err != nil {
				t.Fatalf("decodeAgentChatReply error: %v", err)
			}
			if strings.TrimSpace(got) == "" {
				t.Fatalf("decodeAgentChatReply(%s) returned empty text", body)
			}
		})
	}
}

func TestTasksResubscribeStreamsCurrentAndSubsequentEvents(t *testing.T) {
	d := newDispatcher()
	initial := &Task{ID: "task-1", ContextID: "ctx-1", Kind: "task", Status: TaskStatus{State: stateWorking, Timestamp: time.Now().UTC().Format(time.RFC3339)}}
	d.store(initial)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tasks/resubscribe","params":{"id":"task-1"}}`)).WithContext(ctx)
	rw := newFlushRecorder()
	done := make(chan struct{})
	go func() {
		d.serve(rw, req, nil)
		close(done)
	}()

	first := rw.next(t)
	if first.Result.ID != initial.ID || first.Result.Status.State != stateWorking {
		t.Fatalf("first resubscribe event = %+v, want current working task", first.Result)
	}

	final := &Task{ID: "task-1", ContextID: "ctx-1", Kind: "task", Status: TaskStatus{State: stateCompleted, Timestamp: time.Now().UTC().Format(time.RFC3339)}, Artifacts: []Artifact{textArtifact("done")}}
	d.store(final)
	second := rw.next(t)
	if second.Result.ID != final.ID || second.Result.Status.State != stateCompleted || textOf(second.Result.Artifacts[0].Parts) != "done" {
		t.Fatalf("second resubscribe event = %+v, want completed update", second.Result)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("resubscribe did not return after terminal update")
	}
}

func TestInputRequiredErrorCreatesContinuableTask(t *testing.T) {
	d := newDispatcher()
	first := rpcTaskFromBody(t, d, `{
		"jsonrpc":"2.0","id":1,"method":"message/send",
		"params":{"message":{"role":"user","kind":"message","messageId":"m1",
			"parts":[{"kind":"text","text":"start approval"}]}}}`, func(_ context.Context, text string) (string, error) {
		return "", errors.New("agent run run-1 paused for approval: waiting for operator")
	})
	if first.Status.State != stateInputRequired {
		t.Fatalf("state = %q, want input-required", first.Status.State)
	}
	if textOf(first.Artifacts[0].Parts) != "agent run run-1 paused for approval: waiting for operator" {
		t.Fatalf("artifact = %+v, want handoff message", first.Artifacts)
	}

	body := fmt.Sprintf(`{
		"jsonrpc":"2.0","id":2,"method":"message/send",
		"params":{"message":{"role":"user","kind":"message","messageId":"m2","taskId":"%s","contextId":"%s",
			"parts":[{"kind":"text","text":"approved"}]}}}`, first.ID, first.ContextID)
	continued := rpcTaskFromBody(t, d, body, func(_ context.Context, text string) (string, error) {
		return "continued after " + text, nil
	})
	if continued.ID != first.ID || continued.ContextID != first.ContextID {
		t.Fatalf("continued identity = %s/%s, want %s/%s", continued.ID, continued.ContextID, first.ID, first.ContextID)
	}
	if continued.Status.State != stateCompleted || len(continued.History) != 4 {
		t.Fatalf("continued task = %+v, want completed task with prior input-required history", continued)
	}
	if textOf(continued.History[1].Parts) != "agent run run-1 paused for approval: waiting for operator" || textOf(continued.History[3].Parts) != "continued after approved" {
		t.Fatalf("continued history = %+v", continued.History)
	}
}

type flushRecorder struct {
	*httptest.ResponseRecorder
	ch chan string
}

func newFlushRecorder() *flushRecorder {
	return &flushRecorder{ResponseRecorder: httptest.NewRecorder(), ch: make(chan string, 16)}
}

func (r *flushRecorder) Flush() {
	body := r.Body.String()
	r.Body.Reset()
	for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			r.ch <- line
		}
	}
}

func (r *flushRecorder) next(t *testing.T) struct {
	Result Task      `json:"result"`
	Error  *rpcError `json:"error"`
} {
	t.Helper()
	select {
	case line := <-r.ch:
		line = strings.TrimPrefix(line, "data: ")
		var event struct {
			Result Task      `json:"result"`
			Error  *rpcError `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode event %q: %v", line, err)
		}
		if event.Error != nil {
			t.Fatalf("event error: %+v", event.Error)
		}
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SSE event")
	}
	return struct {
		Result Task      `json:"result"`
		Error  *rpcError `json:"error"`
	}{}
}

func rpcTaskFromDispatcher(t *testing.T, d *dispatcher, id string) Task {
	t.Helper()
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tasks/get","params":{"id":"%s"}}`, id)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	d.serve(rr, req, nil)
	var resp struct {
		Result Task      `json:"result"`
		Error  *rpcError `json:"error"`
	}
	if err := json.NewDecoder(rr.Result().Body).Decode(&resp); err != nil {
		t.Fatalf("decode tasks/get: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("tasks/get error: %+v", resp.Error)
	}
	return resp.Result
}

func rpcTaskFromBody(t *testing.T, d *dispatcher, body string, invoke Invoke) Task {
	t.Helper()
	var resp struct {
		Result Task      `json:"result"`
		Error  *rpcError `json:"error"`
	}
	rpcDispatcher(t, d, body, invoke, &resp)
	if resp.Error != nil {
		t.Fatalf("rpc error: %+v", resp.Error)
	}
	return resp.Result
}

func rpcDispatcher(t *testing.T, d *dispatcher, body string, invoke Invoke, v any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	d.serve(rr, req, invoke)
	if err := json.NewDecoder(rr.Result().Body).Decode(v); err != nil {
		t.Fatalf("decode dispatcher response: %v", err)
	}
}

func TestUnknownMethod(t *testing.T) {
	ts, cleanup := newGatewayWithAgent(t)
	defer cleanup()

	var resp struct {
		Error *rpcError `json:"error"`
	}
	rpc(t, ts.URL+"/agents/echo", `{"jsonrpc":"2.0","id":1,"method":"unknown","params":{}}`, &resp)
	if resp.Error == nil || resp.Error.Code != errMethodNotFound {
		t.Errorf("expected method-not-found, got %+v", resp.Error)
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

func skillIDs(skills []Skill) []string {
	ids := make([]string, 0, len(skills))
	for _, skill := range skills {
		ids = append(ids, skill.ID)
	}
	return ids
}
