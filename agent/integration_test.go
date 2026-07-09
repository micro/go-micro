package agent

import (
	"context"
	"io"
	"strings"
	"testing"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/client"
	codecBytes "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

// fakeGen drives the fake provider's Generate. Tests set it and reset
// it with a deferred cleanup. Tests in this package are not parallel,
// so a package-level hook is safe.
var fakeGen func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error)
var fakeStream func(ctx context.Context, opts ai.Options, req *ai.Request) (ai.Stream, error)

type fakeModel struct{ opts ai.Options }

func (m *fakeModel) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *fakeModel) Options() ai.Options { return m.opts }
func (m *fakeModel) Generate(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	if fakeGen != nil {
		return fakeGen(ctx, m.opts, req)
	}
	return &ai.Response{Reply: "ok"}, nil
}
func (m *fakeModel) Stream(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (ai.Stream, error) {
	if fakeStream != nil {
		return fakeStream(ctx, m.opts, req)
	}
	return &sliceStream{chunks: []string{"ok"}}, nil
}
func (m *fakeModel) String() string { return "fake" }

type sliceStream struct {
	chunks []string
	idx    int
	closed bool
}

func (s *sliceStream) Recv() (*ai.Response, error) {
	if s.idx >= len(s.chunks) {
		return nil, io.EOF
	}
	chunk := s.chunks[s.idx]
	s.idx++
	return &ai.Response{Reply: chunk}, nil
}

func (s *sliceStream) Close() error {
	s.closed = true
	return nil
}

func init() {
	ai.Register("fake", func(opts ...ai.Option) ai.Model {
		m := &fakeModel{}
		_ = m.Init(opts...)
		return m
	})
	ai.RegisterStream("fake")
	ai.RegisterToolStream("fake")
}

// fakeClient embeds the default client (so NewRequest works) and
// overrides Call with a test-supplied function.
type fakeClient struct {
	client.Client
	callFn func(ctx context.Context, req client.Request, rsp interface{}) error
}

func (c *fakeClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	return c.callFn(ctx, req, rsp)
}

func newTestAgent(opts ...Option) *agentImpl {
	base := []Option{
		Provider("fake"),
		WithRegistry(registry.NewMemoryRegistry()),
		WithStore(store.NewMemoryStore()),
	}
	a := New(append(base, opts...)...).(*agentImpl)
	a.setup()
	return a
}

// The model is offered the plan and delegate tools, and calling the
// plan tool persists the plan to memory.
func TestAskExposesAndRunsPlan(t *testing.T) {
	var sawPlan, sawDelegate bool
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		for _, tl := range req.Tools {
			switch tl.Name {
			case toolPlan:
				sawPlan = true
			case toolDelegate:
				sawDelegate = true
			}
		}
		// Simulate the model recording a plan.
		if opts.ToolHandler != nil {
			opts.ToolHandler(context.Background(), ai.ToolCall{
				Name: toolPlan,
				Input: map[string]any{
					"steps": []any{map[string]any{"task": "step one", "status": "pending"}},
				},
			})
		}
		return &ai.Response{Answer: "done"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("worker"))
	resp, err := a.Ask(context.Background(), "do some multi-step work")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !sawPlan || !sawDelegate {
		t.Errorf("model should be offered plan and delegate tools: plan=%v delegate=%v", sawPlan, sawDelegate)
	}
	if resp.Reply == "" {
		t.Error("Ask returned empty reply")
	}
	if plan := a.loadPlan(); !strings.Contains(plan, "step one") {
		t.Errorf("plan tool result not persisted; loadPlan() = %q", plan)
	}
}

// Delegating with no matching agent creates an ephemeral sub-agent with
// a fresh, isolated context (no builtin tools) and returns its reply.
func TestDelegateEphemeral(t *testing.T) {
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if strings.Contains(req.SystemPrompt, "sub-agent") {
			for _, tl := range req.Tools {
				if tl.Name == toolPlan || tl.Name == toolDelegate {
					t.Errorf("ephemeral sub-agent must not have builtin tool %q", tl.Name)
				}
			}
			return &ai.Response{Reply: "subtask complete"}, nil
		}
		return &ai.Response{Reply: "parent"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("root"))
	content := a.handleDelegate(context.Background(), ai.ToolCall{Name: "delegate", Input: map[string]any{"task": "summarize the report"}}).Content
	if !strings.Contains(content, "subtask complete") {
		t.Errorf("delegate should return the sub-agent's reply; got %q", content)
	}
}

// Delegating to a name that resolves to a registered agent goes over
// RPC to that agent rather than spawning a sub-agent.
func TestDelegateToRegisteredAgent(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	if err := reg.Register(&registry.Service{
		Name:     "comms",
		Metadata: map[string]string{"type": "agent"},
		Nodes:    []*registry.Node{{Id: "comms-1", Address: "127.0.0.1:0"}},
	}); err != nil {
		t.Fatalf("register agent: %v", err)
	}

	var calledService, calledEndpoint string
	fc := &fakeClient{Client: client.DefaultClient}
	fc.callFn = func(ctx context.Context, req client.Request, rsp interface{}) error {
		calledService, calledEndpoint = req.Service(), req.Endpoint()
		frame := rsp.(*codecBytes.Frame)
		frame.Data = []byte(`{"reply":"notified alice","agent":"comms"}`)
		return nil
	}

	// fakeGen guards against the ephemeral path being taken by mistake.
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		t.Error("delegate to a registered agent must not spawn a sub-agent")
		return &ai.Response{}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("root"), WithRegistry(reg), WithClient(fc))
	content := a.handleDelegate(context.Background(), ai.ToolCall{Name: "delegate", Input: map[string]any{"task": "notify alice", "to": "comms"}}).Content

	if calledService != "comms" || calledEndpoint != "Agent.Chat" {
		t.Errorf("expected RPC to comms Agent.Chat, got %s %s", calledService, calledEndpoint)
	}
	if !strings.Contains(content, "notified alice") {
		t.Errorf("delegate-first result missing agent reply; got %q", content)
	}
}

// Delegate requires a task.
func TestDelegateRequiresTask(t *testing.T) {
	a := newTestAgent(Name("root"))
	content := a.handleDelegate(context.Background(), ai.ToolCall{Name: "delegate", Input: map[string]any{}}).Content
	if !strings.Contains(content, "error") {
		t.Errorf("delegate with no task should error; got %q", content)
	}
}

func TestCompactingMemorySummarizesAndRecallsArchivedContext(t *testing.T) {
	var sawSummary, sawRecall bool
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		for _, msg := range req.Messages {
			text := msg.Content.(string)
			if strings.Contains(text, "Conversation memory summary") && strings.Contains(text, "alpha project") {
				sawSummary = true
			}
			if strings.Contains(text, "alpha project budget is 42") {
				sawRecall = true
			}
		}
		return &ai.Response{Reply: "ok"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("memory"), CompactMemory(4, 2), MemoryRecallLimit(3))
	turns := []string{
		"alpha project budget is 42",
		"beta project owner is sam",
		"gamma project deadline is monday",
		"delta project status is green",
		"epsilon project risk is low",
	}
	for _, turn := range turns {
		if _, err := a.Ask(context.Background(), turn); err != nil {
			t.Fatalf("Ask(%q): %v", turn, err)
		}
	}
	if got := len(a.mem.Messages()); got > 4 {
		t.Fatalf("compacted memory retained %d messages, want <= 4", got)
	}
	if _, err := a.Ask(context.Background(), "what was the alpha budget?"); err != nil {
		t.Fatalf("Ask recall: %v", err)
	}
	if !sawSummary {
		t.Error("model request did not include a deterministic compacted summary")
	}
	if !sawRecall {
		t.Error("model request did not recall archived matching context")
	}
}
