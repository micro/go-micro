package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/client"
	codecBytes "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

func TestResumeCompletedCheckpointDoesNotReplayModel(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "durable-agent")
	calls := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		calls++
		return &ai.Response{Reply: "done"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("durable-agent"), WithCheckpoint(cp))
	resp, err := a.Ask(ctx, "finish the work")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if calls != 1 {
		t.Fatalf("model calls after Ask = %d, want 1", calls)
	}

	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		calls++
		t.Fatal("Resume of a completed run replayed the model")
		return nil, nil
	}
	resumed, err := Resume(ctx, a, resp.RunID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resumed.Reply != "done" {
		t.Fatalf("resumed reply = %q, want done", resumed.Reply)
	}
	if resumed.RunID != resp.RunID {
		t.Fatalf("resumed run id = %q, want %q", resumed.RunID, resp.RunID)
	}
	if calls != 1 {
		t.Fatalf("model calls after Resume = %d, want 1", calls)
	}
}

func TestResumeFailedCheckpointDoesNotReplayCompletedTool(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "tool-resume-agent")
	toolRuns := 0
	first := true
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if opts.ToolHandler != nil {
			res := opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "external.charge", Input: map[string]any{"order": "42"}})
			if res.Content != "charged" {
				t.Fatalf("tool result = %q, want charged", res.Content)
			}
		}
		if first {
			first = false
			return nil, errors.New("model connection dropped after tool")
		}
		return &ai.Response{Reply: "finished from checkpoint"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("tool-resume-agent"), WithCheckpoint(cp),
		WithTool("external.charge", "charge once", nil, func(context.Context, map[string]any) (string, error) {
			toolRuns++
			return "charged", nil
		}))
	_, err := a.Ask(ctx, "charge order 42")
	if err == nil {
		t.Fatal("Ask succeeded, want simulated failure")
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions after failed Ask = %d, want 1", toolRuns)
	}

	runs, err := Pending(ctx, a)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("Pending returned %d runs, want 1", len(runs))
	}
	resp, err := Resume(ctx, a, runs[0].ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resp.Reply != "finished from checkpoint" {
		t.Fatalf("Resume reply = %q", resp.Reply)
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions after Resume = %d, want completed tool was not replayed", toolRuns)
	}
}

func TestCheckpointSkipsDuplicateToolWithinAsk(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "tool-dedupe-agent")
	toolRuns := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if opts.ToolHandler == nil {
			t.Fatal("missing tool handler")
		}
		opts.ToolHandler(ctx, ai.ToolCall{ID: "plan-1", Name: toolPlan, Input: map[string]any{
			"steps": []any{
				map[string]any{"task": "create Design task", "status": "pending"},
			},
		}})
		for i := 0; i < 3; i++ {
			res := opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "external.create", Input: map[string]any{"title": "Design"}})
			if res.Content != "created Design" {
				t.Fatalf("tool result %d = %q, want cached created Design", i, res.Content)
			}
		}
		return &ai.Response{Reply: "done"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("tool-dedupe-agent"), WithCheckpoint(cp),
		WithTool("external.create", "create once", nil, func(context.Context, map[string]any) (string, error) {
			toolRuns++
			return "created Design", nil
		}))
	if _, err := a.Ask(ctx, "create Design once"); err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions = %d, want duplicate calls within the run replayed from checkpoint", toolRuns)
	}
	if plan := a.loadPlan(); !strings.Contains(plan, `"status":"done"`) {
		t.Fatalf("plan = %s, want completed action marked done", plan)
	}
}

func TestCheckpointContinuesRunWithUnfinishedPlanStep(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "unfinished-plan-agent")

	reg := registry.NewMemoryRegistry()
	if err := reg.Register(&registry.Service{
		Name:     "comms",
		Metadata: map[string]string{"type": "agent"},
		Nodes:    []*registry.Node{{Id: "comms-1", Address: "127.0.0.1:0"}},
	}); err != nil {
		t.Fatalf("register comms agent: %v", err)
	}

	delegateCalls := 0
	fc := &fakeClient{Client: client.DefaultClient}
	fc.callFn = func(ctx context.Context, req client.Request, rsp interface{}) error {
		delegateCalls++
		if req.Service() != "comms" || req.Endpoint() != "Agent.Chat" {
			t.Fatalf("delegate RPC = %s %s, want comms Agent.Chat", req.Service(), req.Endpoint())
		}
		frame := rsp.(*codecBytes.Frame)
		frame.Data = []byte(`{"reply":"owner notified","agent":"comms"}`)
		return nil
	}

	modelCalls := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		modelCalls++
		if opts.ToolHandler == nil {
			t.Fatal("missing tool handler")
		}
		switch modelCalls {
		case 1:
			opts.ToolHandler(ctx, ai.ToolCall{ID: "plan-1", Name: toolPlan, Input: map[string]any{
				"steps": []any{
					map[string]any{"task": "create launch tasks", "status": "done"},
					map[string]any{"task": "delegate readiness notification to comms", "status": "in_progress"},
				},
			}})
			return &ai.Response{Reply: "tasks are ready"}, nil
		case 2:
			if !strings.Contains(req.Prompt, "delegate readiness notification to comms") {
				t.Fatalf("continuation prompt = %q, want unfinished step", req.Prompt)
			}
			res := opts.ToolHandler(ctx, ai.ToolCall{ID: "delegate-1", Name: toolDelegate, Input: map[string]any{"task": "Notify owner@acme.com that the launch plan is ready", "to": "comms"}})
			if !strings.Contains(res.Content, "owner notified") {
				t.Fatalf("delegate result = %q, want owner notified", res.Content)
			}
			return &ai.Response{Reply: "all done"}, nil
		default:
			t.Fatalf("unexpected model call %d", modelCalls)
			return nil, nil
		}
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("unfinished-plan-agent"), WithCheckpoint(cp), WithRegistry(reg), WithClient(fc))
	resp, err := a.Ask(ctx, "create tasks and notify owner")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if resp.Reply != "all done" {
		t.Fatalf("reply = %q, want final continuation reply", resp.Reply)
	}
	if modelCalls != 2 {
		t.Fatalf("model calls = %d, want initial plus continuation", modelCalls)
	}
	if delegateCalls != 1 {
		t.Fatalf("delegate calls = %d, want exactly one", delegateCalls)
	}
	if unfinished := a.unfinishedPlanSteps(); len(unfinished) != 0 {
		t.Fatalf("unfinished plan steps = %v, want none", unfinished)
	}
}

func TestCheckpointContinuesRunThroughSeveralSingleStepTurns(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "single-step-plan-agent")

	completed := []string{}
	modelCalls := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		modelCalls++
		if opts.ToolHandler == nil {
			t.Fatal("missing tool handler")
		}
		switch modelCalls {
		case 1:
			opts.ToolHandler(ctx, ai.ToolCall{ID: "plan-1", Name: toolPlan, Input: map[string]any{
				"steps": []any{
					map[string]any{"task": "create Design task", "status": "pending"},
					map[string]any{"task": "create Build task", "status": "pending"},
					map[string]any{"task": "create Ship task", "status": "pending"},
					map[string]any{"task": "delegate readiness notification", "status": "pending"},
				},
			}})
			return &ai.Response{Reply: "planned"}, nil
		case 2, 3, 4, 5:
			want := []string{"create Design task", "create Build task", "create Ship task", "delegate readiness notification"}[modelCalls-2]
			if !strings.Contains(req.Prompt, want) {
				t.Fatalf("continuation prompt %d = %q, want %q", modelCalls, req.Prompt, want)
			}
			res := opts.ToolHandler(ctx, ai.ToolCall{ID: want, Name: "external.step", Input: map[string]any{"step": want}})
			if res.Content != "completed "+want {
				t.Fatalf("tool result = %q, want completed %s", res.Content, want)
			}
			if modelCalls == 5 {
				return &ai.Response{Reply: "all plan steps complete"}, nil
			}
			return &ai.Response{Reply: "one more step complete"}, nil
		default:
			t.Fatalf("unexpected model call %d", modelCalls)
			return nil, nil
		}
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("single-step-plan-agent"), WithCheckpoint(cp),
		WithTool("external.step", "complete one planned step", nil, func(ctx context.Context, input map[string]any) (string, error) {
			step, _ := input["step"].(string)
			completed = append(completed, step)
			return "completed " + step, nil
		}))
	resp, err := a.Ask(ctx, "work through the launch plan")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if resp.Reply != "all plan steps complete" {
		t.Fatalf("reply = %q, want final continuation reply", resp.Reply)
	}
	if modelCalls != 5 {
		t.Fatalf("model calls = %d, want initial plus four continuations", modelCalls)
	}
	if len(completed) != 4 {
		t.Fatalf("completed steps = %v, want four tool-backed continuations", completed)
	}
	if unfinished := a.unfinishedPlanSteps(); len(unfinished) != 0 {
		t.Fatalf("unfinished plan steps = %v, want none", unfinished)
	}
}

func TestResumeFailedCheckpointAfterFreshAgentRestart(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	cp := flow.StoreCheckpoint(st, "restart-resume-agent")
	toolRuns := 0
	modelCalls := 0
	failFirst := true
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		modelCalls++
		if opts.ToolHandler != nil {
			res := opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "external.provision", Input: map[string]any{"service": "api"}})
			if res.Content != "provisioned" {
				t.Fatalf("tool result = %q, want provisioned", res.Content)
			}
		}
		if failFirst {
			failFirst = false
			return nil, errors.New("process stopped after tool checkpoint")
		}
		return &ai.Response{Reply: "resumed after restart"}, nil
	}
	defer func() { fakeGen = nil }()

	newAgent := func() *agentImpl {
		return newTestAgent(Name("restart-resume-agent"), WithStore(st), WithCheckpoint(cp),
			WithTool("external.provision", "provision service once", nil, func(context.Context, map[string]any) (string, error) {
				toolRuns++
				return "provisioned", nil
			}))
	}

	first := newAgent()
	_, err := first.Ask(ctx, "provision api")
	if err == nil {
		t.Fatal("Ask succeeded, want simulated process stop")
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions after failed Ask = %d, want 1", toolRuns)
	}
	runs, err := Pending(ctx, first)
	if err != nil {
		t.Fatalf("Pending before restart: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("Pending before restart returned %d runs, want 1", len(runs))
	}
	summaries, err := ListRunSummaries(st, "restart-resume-agent")
	if err != nil {
		t.Fatalf("ListRunSummaries before restart: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("run summaries before restart = %d, want 1", len(summaries))
	}
	if summaries[0].RunID != runs[0].ID || summaries[0].Status != "error" || summaries[0].Checkpoint != "failed" || summaries[0].Stage != agentAskStep {
		t.Fatalf("summary before restart = %#v, want failed ask checkpoint for %s", summaries[0], runs[0].ID)
	}
	if summaries[0].Events < 4 || summaries[0].LastError == "" {
		t.Fatalf("summary before restart lacks debug history/error: %#v", summaries[0])
	}

	restarted := newAgent()
	resp, err := Resume(ctx, restarted, runs[0].ID)
	if err != nil {
		t.Fatalf("Resume after restart: %v", err)
	}
	if resp.Reply != "resumed after restart" || resp.RunID != runs[0].ID {
		t.Fatalf("response = %#v, want resumed reply on original run id", resp)
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions after restart resume = %d, want checkpointed tool not replayed", toolRuns)
	}
	if modelCalls != 2 {
		t.Fatalf("model calls = %d, want initial call plus resumed call", modelCalls)
	}
	loaded, ok, err := cp.Load(ctx, runs[0].ID)
	if err != nil || !ok {
		t.Fatalf("Load resumed run ok=%v err=%v", ok, err)
	}
	if loaded.Status != "done" || loaded.ParentID != runs[0].ParentID {
		t.Fatalf("loaded run status/parent = %s/%s, want done/%s", loaded.Status, loaded.ParentID, runs[0].ParentID)
	}
	summaries, err = ListRunSummaries(st, "restart-resume-agent")
	if err != nil {
		t.Fatalf("ListRunSummaries after restart: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("run summaries after restart = %d, want 1", len(summaries))
	}
	if summaries[0].RunID != runs[0].ID || summaries[0].Status != "done" || summaries[0].Checkpoint != "done" || summaries[0].Stage != agentAskStep {
		t.Fatalf("summary after restart = %#v, want done ask checkpoint for %s", summaries[0], runs[0].ID)
	}
	if summaries[0].Events < 7 {
		t.Fatalf("summary after restart recorded %d events, want durable failure/resume/done history", summaries[0].Events)
	}
	events, err := LoadRunEvents(st, "restart-resume-agent", runs[0].ID)
	if err != nil {
		t.Fatalf("LoadRunEvents after restart: %v", err)
	}
	seen := map[string]bool{"run": false, "tool": false, "checkpoint": false, "error": false, "resume": false, "done": false}
	for _, e := range events {
		if _, ok := seen[e.Kind]; ok {
			seen[e.Kind] = true
		}
	}
	for kind, ok := range seen {
		if !ok {
			t.Fatalf("events after restart missing %s: %#v", kind, events)
		}
	}
}

func TestResumePendingAfterFreshAgentRestartDoesNotReplayCompletedTool(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	cp := flow.StoreCheckpoint(st, "startup-resume-agent")
	toolRuns := 0
	failFirst := true
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if opts.ToolHandler != nil {
			res := opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "external.allocate", Input: map[string]any{"cluster": "blue"}})
			if res.Content != "allocated" {
				t.Fatalf("tool result = %q, want allocated", res.Content)
			}
		}
		if failFirst {
			failFirst = false
			return nil, errors.New("process stopped before final response")
		}
		return &ai.Response{Reply: "startup recovery complete"}, nil
	}
	defer func() { fakeGen = nil }()

	newAgent := func() *agentImpl {
		return newTestAgent(Name("startup-resume-agent"), WithStore(st), WithCheckpoint(cp),
			WithTool("external.allocate", "allocate capacity once", nil, func(context.Context, map[string]any) (string, error) {
				toolRuns++
				return "allocated", nil
			}))
	}

	first := newAgent()
	_, err := first.Ask(ctx, "allocate blue capacity")
	if err == nil {
		t.Fatal("Ask succeeded, want simulated process stop")
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions after failed Ask = %d, want 1", toolRuns)
	}

	restarted := newAgent()
	failedRun, err := ResumePending(ctx, restarted)
	if err != nil {
		t.Fatalf("ResumePending after restart: failedRun=%q err=%v", failedRun, err)
	}
	if failedRun != "" {
		t.Fatalf("failed run = %q, want none", failedRun)
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions after ResumePending = %d, want completed tool not replayed", toolRuns)
	}
	runs, err := Pending(ctx, restarted)
	if err != nil {
		t.Fatalf("Pending after ResumePending: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("Pending after ResumePending = %#v, want none", runs)
	}
	summaries, err := ListRunSummaries(st, "startup-resume-agent")
	if err != nil {
		t.Fatalf("ListRunSummaries after ResumePending: %v", err)
	}
	if len(summaries) != 1 || summaries[0].Status != "done" || summaries[0].Checkpoint != "done" {
		t.Fatalf("summary after ResumePending = %#v, want one done run", summaries)
	}
}

func TestResumeFailedCheckpointDoesNotDuplicateCompactedMemory(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	cp := flow.StoreCheckpoint(st, "memory-resume-agent")
	failRetry := true
	var sawRecall bool
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		for _, msg := range req.Messages {
			if text, ok := msg.Content.(string); ok && strings.Contains(text, "alpha code is 42") {
				sawRecall = true
			}
		}
		if strings.Contains(req.Prompt, "use alpha code") && failRetry {
			failRetry = false
			return nil, errors.New("model connection dropped")
		}
		return &ai.Response{Reply: "ok"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("memory-resume-agent"), WithStore(st), WithCheckpoint(cp), CompactMemory(4, 1), MemoryRecallLimit(2))
	for _, msg := range []string{"alpha code is 42", "beta note", "gamma note"} {
		if _, err := a.Ask(ctx, msg); err != nil {
			t.Fatalf("Ask(%q): %v", msg, err)
		}
	}

	_, err := a.Ask(ctx, "use alpha code now")
	if err == nil {
		t.Fatal("Ask succeeded, want simulated provider failure")
	}
	if got := countMemoryContent(a.mem.Messages(), "use alpha code now"); got != 1 {
		t.Fatalf("failed Ask stored prompt %d times, want 1", got)
	}

	runs, err := Pending(ctx, a)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("Pending returned %d runs, want 1", len(runs))
	}
	if _, err := Resume(ctx, a, runs[0].ID); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if got := countMemoryContent(a.mem.Messages(), "use alpha code now"); got != 1 {
		t.Fatalf("resumed failed Ask stored prompt %d times, want no duplicate", got)
	}
	if !sawRecall {
		t.Fatal("resume did not retrieve archived compacted memory")
	}
	if got := len(a.mem.Messages()); got > 4 {
		t.Fatalf("compacted memory retained %d messages after resume, want <= 4", got)
	}
}

func countMemoryContent(messages []ai.Message, needle string) int {
	var count int
	for _, msg := range messages {
		if text, ok := msg.Content.(string); ok && strings.Contains(text, needle) {
			count++
		}
	}
	return count
}

func TestResumePendingResumesOldestAgentRunsUntilFailure(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "resume-pending-agent")
	base := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	for _, run := range []flow.Run{
		{ID: "run-ok", Flow: "resume-pending-agent", Status: "failed", State: flow.State{Stage: agentAskStep, Data: []byte("ok")}, Started: base},
		{ID: "run-blocked", Flow: "resume-pending-agent", Status: "failed", State: flow.State{Stage: agentAskStep, Data: []byte("block")}, Started: base.Add(time.Minute)},
		{ID: "run-later", Flow: "resume-pending-agent", Status: "failed", State: flow.State{Stage: agentAskStep, Data: []byte("later")}, Started: base.Add(2 * time.Minute)},
	} {
		if err := cp.Save(ctx, run); err != nil {
			t.Fatalf("Save(%s): %v", run.ID, err)
		}
	}

	var prompts []string
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		prompts = append(prompts, req.Prompt)
		if req.Prompt == "block" {
			return nil, errors.New("still blocked")
		}
		return &ai.Response{Reply: req.Prompt + " resumed"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("resume-pending-agent"), WithCheckpoint(cp))
	failedRun, err := ResumePending(ctx, a)
	if err == nil {
		t.Fatal("ResumePending succeeded, want blocked run error")
	}
	if failedRun != "run-blocked" {
		t.Fatalf("failed run = %q, want run-blocked", failedRun)
	}
	if got, want := strings.Join(prompts, ","), "ok,block"; got != want {
		t.Fatalf("prompts = %q, want %q", got, want)
	}
	loaded, ok, err := cp.Load(ctx, "run-ok")
	if err != nil || !ok || loaded.Status != "done" {
		t.Fatalf("run-ok loaded=%v err=%v status=%q, want done", ok, err, loaded.Status)
	}
	loaded, ok, err = cp.Load(ctx, "run-later")
	if err != nil || !ok || loaded.Status != "failed" {
		t.Fatalf("run-later loaded=%v err=%v status=%q, want still failed", ok, err, loaded.Status)
	}
}

func TestPendingReturnsUnfinishedAgentRuns(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "pending-agent")
	run := flow.Run{ID: "run-1", Flow: "pending-agent", Status: "failed", State: flow.State{Stage: agentAskStep, Data: []byte("retry me")}}
	if err := cp.Save(ctx, run); err != nil {
		t.Fatalf("Save: %v", err)
	}
	a := newTestAgent(Name("pending-agent"), WithCheckpoint(cp))
	runs, err := Pending(ctx, a)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(runs) != 1 || runs[0].ID != "run-1" {
		t.Fatalf("Pending = %#v, want run-1", runs)
	}
}

func TestPendingSkipsTerminalCanceledAndExpiredAgentRuns(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "terminal-agent")
	for _, run := range []flow.Run{
		{ID: "active", Flow: "terminal-agent", Status: "failed", State: flow.State{Stage: agentAskStep, Data: []byte("retry me")}},
		{ID: "done", Flow: "terminal-agent", Status: "done", State: flow.State{Stage: agentAskStep, Data: []byte("done")}},
		{ID: "canceled", Flow: "terminal-agent", Status: "canceled", State: flow.State{Stage: agentAskStep, Data: []byte("canceled")}},
		{ID: "expired", Flow: "terminal-agent", Status: "expired", State: flow.State{Stage: agentAskStep, Data: []byte("expired")}},
	} {
		if err := cp.Save(ctx, run); err != nil {
			t.Fatalf("Save(%s): %v", run.ID, err)
		}
	}

	a := newTestAgent(Name("terminal-agent"), WithCheckpoint(cp))
	runs, err := Pending(ctx, a)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(runs) != 1 || runs[0].ID != "active" {
		t.Fatalf("Pending = %#v, want only active failed run", runs)
	}
	for _, id := range []string{"canceled", "expired"} {
		if _, err := Resume(ctx, a, id); err == nil || !strings.Contains(err.Error(), "terminal") {
			t.Fatalf("Resume(%s) err = %v, want terminal status error", id, err)
		}
	}
}

func TestHumanInputPauseResumesSameRunWithInput(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "input-agent")
	calls := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		calls++
		if calls == 1 {
			if opts.ToolHandler != nil {
				opts.ToolHandler(ctx, ai.ToolCall{ID: "input-1", Name: toolHumanInput, Input: map[string]any{"prompt": "Which region should I deploy to?"}})
			}
			return &ai.Response{Reply: "waiting"}, nil
		}
		if !strings.Contains(req.Prompt, "Human input: us-east-1") {
			t.Fatalf("resumed prompt = %q, want human input", req.Prompt)
		}
		return &ai.Response{Reply: "deploying to us-east-1"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("input-agent"), WithCheckpoint(cp))
	_, err := a.Ask(ctx, "deploy the service")
	if err == nil {
		t.Fatal("Ask succeeded, want input-required pause")
	}
	runs, err := Pending(ctx, a)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != "paused" || runs[0].State.Stage != agentInputStep {
		t.Fatalf("paused runs = %#v, want one input-required run", runs)
	}
	var pause inputPause
	if err := runs[0].State.Scan(&pause); err != nil {
		t.Fatalf("Scan pause: %v", err)
	}
	if pause.OriginalMessage != "deploy the service" || pause.Prompt != "Which region should I deploy to?" {
		t.Fatalf("pause = %#v", pause)
	}

	if _, err := Resume(ctx, a, runs[0].ID); err == nil || !strings.Contains(err.Error(), "ResumeInput") {
		t.Fatalf("Resume input-required err = %v, want guidance", err)
	}
	resp, err := ResumeInput(ctx, a, runs[0].ID, "us-east-1")
	if err != nil {
		t.Fatalf("ResumeInput: %v", err)
	}
	if resp.RunID != runs[0].ID || resp.Reply != "deploying to us-east-1" {
		t.Fatalf("response = %#v", resp)
	}
	loaded, ok, err := cp.Load(ctx, runs[0].ID)
	if err != nil || !ok {
		t.Fatalf("Load resumed run ok=%v err=%v", ok, err)
	}
	if loaded.Status != "done" {
		t.Fatalf("resumed run status = %q, want done", loaded.Status)
	}
}

func TestHumanInputResumeHonorsCanceledContextAndLeavesRunPending(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "input-cancel-agent")
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if opts.ToolHandler != nil {
			opts.ToolHandler(ctx, ai.ToolCall{ID: "input-1", Name: toolHumanInput, Input: map[string]any{"prompt": "Approve deploy?"}})
		}
		return &ai.Response{Reply: "waiting"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("input-cancel-agent"), WithCheckpoint(cp))
	if _, err := a.Ask(ctx, "deploy the service"); err == nil {
		t.Fatal("Ask succeeded, want input-required pause")
	}
	runs, err := Pending(ctx, a)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("Pending returned %d runs, want 1: %#v", len(runs), runs)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := ResumeInput(canceled, a, runs[0].ID, "yes"); !errors.Is(err, context.Canceled) {
		t.Fatalf("ResumeInput canceled err = %v, want context.Canceled", err)
	}

	loaded, ok, err := cp.Load(ctx, runs[0].ID)
	if err != nil || !ok {
		t.Fatalf("Load paused run ok=%v err=%v", ok, err)
	}
	if loaded.Status != "paused" || loaded.State.Stage != agentInputStep {
		t.Fatalf("run status/stage after canceled resume = %s/%s, want paused/%s", loaded.Status, loaded.State.Stage, agentInputStep)
	}
	var pause inputPause
	if err := loaded.State.Scan(&pause); err != nil {
		t.Fatalf("Scan pause after canceled resume: %v", err)
	}
	if pause.OriginalMessage != "deploy the service" || pause.Prompt != "Approve deploy?" {
		t.Fatalf("pause after canceled resume = %#v", pause)
	}
}

func TestApprovalDenialPausesCheckpointedRunAndResumeContinues(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "approval-agent")
	calls := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		calls++
		if opts.ToolHandler != nil {
			opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "external.approve", Input: map[string]any{"id": "42"}})
		}
		return &ai.Response{Reply: "model saw approval result"}, nil
	}
	defer func() { fakeGen = nil }()

	approved := false
	a := newTestAgent(Name("approval-agent"), WithCheckpoint(cp),
		WithTool("external.approve", "guarded external action", nil, func(context.Context, map[string]any) (string, error) { return "ok", nil }),
		ApproveTool(func(tool string, input map[string]any) (bool, string) {
			return approved, "waiting for operator"
		}))
	_, err := a.Ask(ctx, "send the guarded update")
	if err == nil {
		t.Fatal("Ask succeeded, want paused approval error")
	}

	runs, err := Pending(ctx, a)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("Pending returned %d runs, want 1: %#v", len(runs), runs)
	}
	if runs[0].Status != "paused" || runs[0].State.Stage != agentApprovalStep {
		t.Fatalf("run status/stage = %s/%s, want paused/%s", runs[0].Status, runs[0].State.Stage, agentApprovalStep)
	}
	if got := string(runs[0].State.Data); got != "send the guarded update" {
		t.Fatalf("paused run data = %q", got)
	}

	approved = true
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		calls++
		if opts.ToolHandler != nil {
			res := opts.ToolHandler(ctx, ai.ToolCall{ID: "call-2", Name: "external.approve", Input: map[string]any{"id": "42"}})
			if res.Refused != "" {
				t.Fatalf("resumed call was refused: %#v", res)
			}
		}
		return &ai.Response{Reply: "done after approval"}, nil
	}
	resp, err := Resume(ctx, a, runs[0].ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resp.Reply != "done after approval" {
		t.Fatalf("Resume reply = %q", resp.Reply)
	}
	loaded, ok, err := cp.Load(ctx, runs[0].ID)
	if err != nil || !ok {
		t.Fatalf("Load resumed run ok=%v err=%v", ok, err)
	}
	if loaded.Status != "done" {
		t.Fatalf("resumed run status = %q, want done", loaded.Status)
	}
	if calls != 2 {
		t.Fatalf("model calls = %d, want 2", calls)
	}
}
