package main

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
	"go-micro.dev/v6/service"
	"go-micro.dev/v6/store"
)

// waitForService polls the registry until name is registered, instead of
// sleeping. Keeps the test deterministic.
func waitForService(t *testing.T, reg registry.Registry, name string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if svcs, err := reg.GetService(name); err == nil && len(svcs) > 0 && len(svcs[0].Nodes) > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("service %q never registered", name)
}

// TestPlanDelegateEndToEnd runs the whole feature against the real stack
// — real services, a shared in-memory registry, real RPC, the real agent
// loop, real store — with only the LLM mocked. No mDNS, no sleeps.
func TestPlanDelegateEndToEnd(t *testing.T) {
	ai.Register("mock", newMock)

	// Shared infrastructure: one in-memory registry, a client bound to
	// it, and an in-memory store. Everything resolves through these.
	reg := registry.NewMemoryRegistry()
	cl := client.NewClient(
		client.Registry(reg),
		client.Selector(selector.NewSelector(selector.Registry(reg))),
	)
	mem := store.NewMemoryStore()

	// Real services on the shared registry/client.
	taskSvc := new(TaskService)
	task := service.New(service.Name("task"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	if err := task.Handle(taskSvc); err != nil {
		t.Fatalf("handle task: %v", err)
	}
	if err := task.Start(); err != nil {
		t.Fatalf("start task: %v", err)
	}
	defer task.Stop()

	notifySvc := new(NotifyService)
	notify := service.New(service.Name("notify"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	if err := notify.Handle(notifySvc); err != nil {
		t.Fatalf("handle notify: %v", err)
	}
	if err := notify.Start(); err != nil {
		t.Fatalf("start notify: %v", err)
	}
	defer notify.Stop()

	// Real comms agent (owns notify), registered so delegate reaches it over RPC.
	comms := agent.New(
		agent.Name("comms"),
		agent.Address("127.0.0.1:0"),
		agent.Services("notify"),
		agent.Prompt(commsPrompt),
		agent.Provider("mock"),
		agent.WithRegistry(reg),
		agent.WithClient(cl),
		agent.WithStore(mem),
	)
	go comms.Run()
	defer comms.Stop()

	waitForService(t, reg, "task")
	waitForService(t, reg, "notify")
	waitForService(t, reg, "comms")

	// Real conductor agent (owns task), driven programmatically.
	conductor := agent.New(
		agent.Name("conductor"),
		agent.Address("127.0.0.1:0"),
		agent.Services("task"),
		agent.Prompt("Plan first, create tasks, delegate notifications to the comms agent."),
		agent.Provider("mock"),
		agent.WithRegistry(reg),
		agent.WithClient(cl),
		agent.WithStore(mem),
	)

	resp, err := conductor.Ask(context.Background(),
		"Create three launch tasks: Design, Build, and Ship. Then notify owner@acme.com that the plan is ready.")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if resp.Reply == "" {
		t.Error("conductor returned an empty reply")
	}

	// Tasks were created via real RPC into the task service.
	if n := taskSvc.count(); n != 3 {
		t.Errorf("task service has %d tasks, want 3", n)
	}

	// The plan was persisted to the real store, in the agent's scoped table.
	if recs, err := store.Scope(mem, "agent", "conductor").Read("plan"); err != nil || len(recs) == 0 {
		t.Errorf("plan not persisted to store: err=%v recs=%d", err, len(recs))
	}

	// Delegation reached the comms agent over RPC, which called notify.
	if n := notifySvc.count(); n != 1 {
		t.Errorf("notify service called %d times, want 1 (delegation did not reach comms)", n)
	}
}

// TestFlowDispatchesToAgentEndToEnd proves "Flow triggers, Agent reasons":
// a workflow event hands off to the registered conductor agent, which then
// plans, creates tasks, and delegates to comms — all over real RPC. Only
// the LLM is mocked.
func TestFlowDispatchesToAgentEndToEnd(t *testing.T) {
	ai.Register("mock", newMock)

	reg := registry.NewMemoryRegistry()
	cl := client.NewClient(
		client.Registry(reg),
		client.Selector(selector.NewSelector(selector.Registry(reg))),
	)
	mem := store.NewMemoryStore()

	taskSvc := new(TaskService)
	task := service.New(service.Name("task"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	if err := task.Handle(taskSvc); err != nil {
		t.Fatalf("handle task: %v", err)
	}
	if err := task.Start(); err != nil {
		t.Fatalf("start task: %v", err)
	}
	defer task.Stop()

	notifySvc := new(NotifyService)
	notify := service.New(service.Name("notify"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	if err := notify.Handle(notifySvc); err != nil {
		t.Fatalf("handle notify: %v", err)
	}
	if err := notify.Start(); err != nil {
		t.Fatalf("start notify: %v", err)
	}
	defer notify.Stop()

	comms := agent.New(
		agent.Name("comms"),
		agent.Address("127.0.0.1:0"),
		agent.Services("notify"),
		agent.Prompt(commsPrompt),
		agent.Provider("mock"),
		agent.WithRegistry(reg),
		agent.WithClient(cl),
		agent.WithStore(mem),
	)
	go comms.Run()
	defer comms.Stop()

	// Unlike the previous test, the conductor must be registered (running)
	// so the flow can reach it over RPC.
	conductor := agent.New(
		agent.Name("conductor"),
		agent.Address("127.0.0.1:0"),
		agent.Services("task"),
		agent.Prompt("Plan first, create tasks, delegate notifications to the comms agent."),
		agent.Provider("mock"),
		agent.WithRegistry(reg),
		agent.WithClient(cl),
		agent.WithStore(mem),
	)
	go conductor.Run()
	defer conductor.Stop()

	waitForService(t, reg, "task")
	waitForService(t, reg, "notify")
	waitForService(t, reg, "comms")
	waitForService(t, reg, "conductor")

	// A workflow that hands each event to the conductor agent.
	f := flow.New("onboard",
		flow.Agent("conductor"),
		flow.Prompt("Get the launch ready: {{.Data}}"),
	)
	if err := f.Register(reg, broker.DefaultBroker, cl); err != nil {
		t.Fatalf("flow register: %v", err)
	}

	// Fire the workflow (as a broker event would).
	if err := f.Execute(context.Background(), "three tasks then notify owner@acme.com"); err != nil {
		t.Fatalf("flow execute: %v", err)
	}

	// The flow recorded the agent's reply.
	if rs := f.Results(); len(rs) != 1 || rs[0].Reply == "" {
		t.Errorf("flow result = %+v, want one result with a reply", rs)
	}

	// The agent ran end to end: tasks created, plan stored, comms notified.
	if n := taskSvc.count(); n != 3 {
		t.Errorf("task service has %d tasks, want 3", n)
	}
	if recs, err := store.Scope(mem, "agent", "conductor").Read("plan"); err != nil || len(recs) == 0 {
		t.Errorf("plan not persisted: err=%v recs=%d", err, len(recs))
	}
	if n := notifySvc.count(); n != 1 {
		t.Errorf("notify called %d times, want 1 (flow->agent->delegate->comms chain broken)", n)
	}
}

// TestZeroToHeroContract locks the roadmap's second golden path into the
// ordinary Go test contract. It runs the same executable harness used by
// `make harness`: services + agents + flow + plan/delegate, with only the
// LLM replaced by the deterministic mock provider.
func TestZeroToHeroContract(t *testing.T) {
	if testing.Short() {
		t.Skip("0→hero harness boots an end-to-end system; skipped with -short")
	}
	if err := runPlanDelegate("mock"); err != nil {
		t.Fatalf("0→hero harness: %v", err)
	}
}

func TestPlanDelegateRetriesAfterUnknownDelegateTool(t *testing.T) {
	if testing.Short() {
		t.Skip("0→hero harness boots an end-to-end system; skipped with -short")
	}
	if err := runPlanDelegate("mock-unknown-delegate"); err != nil {
		t.Fatalf("0→hero harness with unknown delegate retry: %v", err)
	}
}

func TestPlanDelegateIdempotentDuplicateNotifyReplay(t *testing.T) {
	if testing.Short() {
		t.Skip("0→hero harness boots an end-to-end system; skipped with -short")
	}
	if err := runPlanDelegate("mock-duplicate-notify"); err != nil {
		t.Fatalf("0→hero harness with duplicate notify replay: %v", err)
	}
}

func TestPlanDelegateIdempotentDuplicateDelegateReplay(t *testing.T) {
	if testing.Short() {
		t.Skip("0→hero harness boots an end-to-end system; skipped with -short")
	}
	if err := runPlanDelegate("mock-duplicate-delegate"); err != nil {
		t.Fatalf("0→hero harness with duplicate delegate replay: %v", err)
	}
}

func TestNotifyServiceDeduplicatesAtlasCloudLaunchReadinessParaphrases(t *testing.T) {
	svc := new(NotifyService)
	variants := []SendRequest{
		{To: "owner at acme dot com", Message: "The launch plan is ready."},
		{To: "launch owner", Message: "Launch readiness is complete."},
		{To: "Owner <owner@acme.com>", Message: "The launch plan is finished and the readiness notification was sent."},
	}
	for _, req := range variants {
		var rsp SendResponse
		if err := svc.Send(context.Background(), &req, &rsp); err != nil {
			t.Fatalf("Send(%+v): %v", req, err)
		}
		if !rsp.Sent {
			t.Fatalf("Send(%+v) returned sent=false", req)
		}
	}
	if got := svc.count(); got != 1 {
		t.Fatalf("notify side effects = %d, want 1 for launch-readiness paraphrase replays", got)
	}
	if got := svc.duplicateAttempts(); got != len(variants)-1 {
		t.Fatalf("duplicate attempts = %d, want %d", got, len(variants)-1)
	}
}

func TestTaskServiceAddIsIdempotentForLaunchTitles(t *testing.T) {
	svc := new(TaskService)
	for _, title := range []string{"Design", "design task", "Build", "Build launch task", "Ship", "ship readiness"} {
		var rsp AddResponse
		if err := svc.Add(context.Background(), &AddRequest{Title: title}, &rsp); err != nil {
			t.Fatalf("Add(%q): %v", title, err)
		}
		if rsp.Task == nil {
			t.Fatalf("Add(%q) returned nil task", title)
		}
	}
	if got := svc.count(); got != 3 {
		t.Fatalf("task count = %d, want 3 after duplicate launch-title replays", got)
	}
}

func TestPlanDelegateExecutionAcceptsDuplicateNotifyReplay(t *testing.T) {
	notifySvc := new(NotifyService)
	for i := 0; i < 2; i++ {
		var rsp SendResponse
		if err := notifySvc.Send(context.Background(), &SendRequest{To: "owner@acme.com", Message: "The launch plan is ready"}, &rsp); err != nil {
			t.Fatalf("Send attempt %d: %v", i+1, err)
		}
	}

	done := make(chan error, 1)
	done <- nil
	if err := waitForPlanDelegateExecution(done, new(TaskService), notifySvc, nil); err != nil {
		t.Fatalf("waitForPlanDelegateExecution returned %v, want duplicate replay accepted", err)
	}
	if got := notifySvc.count(); got != 1 {
		t.Fatalf("notify count = %d, want 1 after duplicate replay", got)
	}
	if got := notifySvc.duplicateAttempts(); got != 1 {
		t.Fatalf("duplicate attempts = %d, want 1 recorded replay", got)
	}
}

func TestPlanDelegateExecutionRecoversUnfinishedPlanAfterPartialTaskSideEffect(t *testing.T) {
	taskSvc := new(TaskService)
	var addRsp AddResponse
	if err := taskSvc.Add(context.Background(), &AddRequest{Title: "Design"}, &addRsp); err != nil {
		t.Fatalf("Add: %v", err)
	}
	notifySvc := new(NotifyService)
	done := make(chan error, 1)
	done <- errors.New("agent run abc has unfinished plan steps: Delegate readiness notification to comms agent")

	recovered := false
	err := waitForPlanDelegateExecution(done, taskSvc, notifySvc, func(ctx context.Context) error {
		recovered = true
		var sendRsp SendResponse
		return notifySvc.Send(ctx, &SendRequest{To: "owner@acme.com", Message: "The launch plan is ready"}, &sendRsp)
	})
	if err != nil {
		t.Fatalf("waitForPlanDelegateExecution returned %v, want partial notify recovery", err)
	}
	if !recovered {
		t.Fatal("missing notify recovery did not run")
	}
	if got := taskSvc.count(); got != 1 {
		t.Fatalf("task count = %d, want completed partial task to stay singular", got)
	}
	if got := notifySvc.count(); got != 1 {
		t.Fatalf("notify count = %d, want recovered notify side effect", got)
	}
}

func TestPlanDelegateExecutionRejectsClaimedCompletionWithoutNotify(t *testing.T) {
	notifySvc := new(NotifyService)
	done := make(chan error, 1)
	done <- nil

	err := waitForPlanDelegateExecution(done, new(TaskService), notifySvc, nil)
	if err == nil {
		t.Fatal("waitForPlanDelegateExecution returned nil, want missing notify side-effect error")
	}
	if got := err.Error(); !strings.Contains(got, "without required notify side effect") {
		t.Fatalf("error = %q, want missing notify side-effect error", got)
	}
}

type scriptedAgent struct {
	replies []func(context.Context, string) (*agent.Response, error)
	calls   int
}

func (a *scriptedAgent) Name() string                                      { return "scripted" }
func (a *scriptedAgent) Init(...agent.Option)                              {}
func (a *scriptedAgent) Options() agent.Options                            { return agent.Options{} }
func (a *scriptedAgent) Stream(context.Context, string) (ai.Stream, error) { return nil, nil }
func (a *scriptedAgent) Run() error                                        { return nil }
func (a *scriptedAgent) Stop() error                                       { return nil }
func (a *scriptedAgent) String() string                                    { return "scripted" }
func (a *scriptedAgent) Ask(ctx context.Context, prompt string) (*agent.Response, error) {
	if a.calls >= len(a.replies) {
		return &agent.Response{Reply: "done"}, nil
	}
	reply := a.replies[a.calls]
	a.calls++
	return reply(ctx, prompt)
}

type failingAgent struct {
	err error
}

func (a failingAgent) Name() string                                         { return "failing" }
func (a failingAgent) Init(...agent.Option)                                 {}
func (a failingAgent) Options() agent.Options                               { return agent.Options{} }
func (a failingAgent) Ask(context.Context, string) (*agent.Response, error) { return nil, a.err }
func (a failingAgent) Stream(context.Context, string) (ai.Stream, error)    { return nil, a.err }
func (a failingAgent) Run() error                                           { return nil }
func (a failingAgent) Stop() error                                          { return nil }
func (a failingAgent) String() string                                       { return "failing" }

func TestRequireConductorPlanRecoversAfterCompletedSideEffects(t *testing.T) {
	mem := store.NewMemoryStore()
	ag := &scriptedAgent{replies: []func(context.Context, string) (*agent.Response, error){
		func(ctx context.Context, prompt string) (*agent.Response, error) {
			if !strings.Contains(prompt, "built-in plan tool") || !strings.Contains(prompt, "Do not call task or notify tools") {
				return nil, errors.New("missing scoped plan recovery prompt")
			}
			if err := store.Scope(mem, "agent", "conductor").Write(&store.Record{Key: "plan", Value: []byte(`{"steps":[{"task":"Design launch task","status":"done"}]}`)}); err != nil {
				return nil, err
			}
			return &agent.Response{Reply: "Plan persisted."}, nil
		},
	}}

	if err := requireConductorPlan(context.Background(), mem, ag); err != nil {
		t.Fatalf("requireConductorPlan returned %v, want recovered plan", err)
	}
	if ag.calls != 1 {
		t.Fatalf("conductor calls = %d, want one plan recovery prompt", ag.calls)
	}
}

func TestRequireConductorPlanFailureNamesScopedRecord(t *testing.T) {
	err := requireConductorPlan(context.Background(), store.NewMemoryStore(), &scriptedAgent{replies: []func(context.Context, string) (*agent.Response, error){
		func(context.Context, string) (*agent.Response, error) {
			return &agent.Response{Reply: "Done without plan."}, nil
		},
	}})
	if err == nil {
		t.Fatal("requireConductorPlan returned nil, want missing plan diagnostic")
	}
	for _, want := range []string{"agent/conductor/plan", "without calling the built-in plan tool"} {
		if got := err.Error(); !strings.Contains(got, want) {
			t.Fatalf("error = %q, want %q", got, want)
		}
	}
}

func TestRequirePersistedPlanBeforeConductorActionsBlocksSideEffects(t *testing.T) {
	mem := store.NewMemoryStore()
	called := false
	wrapped := requirePersistedPlanBeforeConductorActions(mem)(func(context.Context, ai.ToolCall) ai.ToolResult {
		called = true
		return ai.ToolResult{ID: "call-1", Content: `{"ok":true}`}
	})

	res := wrapped(context.Background(), ai.ToolCall{ID: "call-1", Name: "task.Add"})
	if called {
		t.Fatal("side-effecting tool ran before persisted plan")
	}
	if res.Refused != ai.RefusedApproval {
		t.Fatalf("Refused = %q, want %q", res.Refused, ai.RefusedApproval)
	}
	if !strings.Contains(res.Content, "built-in plan tool") {
		t.Fatalf("Content = %q, want plan-first steering message", res.Content)
	}
}

func TestRequirePersistedPlanBeforeConductorActionsAllowsPlanAndPlannedActions(t *testing.T) {
	mem := store.NewMemoryStore()
	var calls []string
	wrapped := requirePersistedPlanBeforeConductorActions(mem)(func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		calls = append(calls, call.Name)
		if call.Name == "plan" {
			if err := store.Scope(mem, "agent", "conductor").Write(&store.Record{Key: "plan", Value: []byte(`{"steps":[{"task":"Design","status":"pending"}]}`)}); err != nil {
				t.Fatalf("write plan: %v", err)
			}
		}
		return ai.ToolResult{ID: call.ID, Content: `{"ok":true}`}
	})

	if res := wrapped(context.Background(), ai.ToolCall{ID: "call-1", Name: "plan"}); res.Refused != "" {
		t.Fatalf("plan Refused = %q, want allowed", res.Refused)
	}
	if res := wrapped(context.Background(), ai.ToolCall{ID: "call-2", Name: "task.Add"}); res.Refused != "" {
		t.Fatalf("planned action Refused = %q, want allowed", res.Refused)
	}
	if want := []string{"plan", "task.Add"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestPlanDelegateConductorRetriesAfterPlanOnlySuccess(t *testing.T) {
	taskSvc := new(TaskService)
	notifySvc := new(NotifyService)
	ag := &scriptedAgent{replies: []func(context.Context, string) (*agent.Response, error){
		func(context.Context, string) (*agent.Response, error) {
			return &agent.Response{Reply: "Plan saved."}, nil
		},
		func(ctx context.Context, prompt string) (*agent.Response, error) {
			if !strings.Contains(prompt, "Execute the persisted plan") {
				return nil, errors.New("missing explicit side-effect recovery prompt")
			}
			for _, title := range []string{"Design", "Build", "Ship"} {
				var rsp AddResponse
				if err := taskSvc.Add(ctx, &AddRequest{Title: title}, &rsp); err != nil {
					return nil, err
				}
			}
			return &agent.Response{Reply: "Tasks created."}, nil
		},
	}}

	step := planDelegateConductorStep(ag, taskSvc, notifySvc)
	if _, err := step(context.Background(), flow.State{}); err != nil {
		t.Fatalf("planDelegateConductorStep returned %v, want plan-only retry success", err)
	}
	if ag.calls != 2 {
		t.Fatalf("conductor calls = %d, want initial plan-only call plus one side-effect retry", ag.calls)
	}
	if got := taskSvc.count(); got != 3 {
		t.Fatalf("task count = %d, want recovered Design/Build/Ship side effects", got)
	}
}

func TestPlanDelegateConductorFailsBeforeNotifyGateAfterPlanOnlyRetryMiss(t *testing.T) {
	step := planDelegateConductorStep(&scriptedAgent{replies: []func(context.Context, string) (*agent.Response, error){
		func(context.Context, string) (*agent.Response, error) {
			return &agent.Response{Reply: "Plan saved."}, nil
		},
		func(context.Context, string) (*agent.Response, error) {
			return &agent.Response{Reply: "Still planning."}, nil
		},
	}}, new(TaskService), new(NotifyService))

	_, err := step(context.Background(), flow.State{})
	if err == nil {
		t.Fatal("planDelegateConductorStep returned nil, want pre-notify-gate task side-effect error")
	}
	for _, want := range []string{"before task side effects completed", "tasks=0/3", "task Add for Design, Build, and Ship"} {
		if got := err.Error(); !strings.Contains(got, want) {
			t.Fatalf("error = %q, want %q", got, want)
		}
	}
}

func TestPlanDelegateConductorAllowsNotifyRecoveryAfterUnfinishedDelegation(t *testing.T) {
	taskSvc := new(TaskService)
	for _, title := range []string{"Design", "Build", "Ship"} {
		var rsp AddResponse
		if err := taskSvc.Add(context.Background(), &AddRequest{Title: title}, &rsp); err != nil {
			t.Fatalf("Add(%q): %v", title, err)
		}
	}
	notifySvc := new(NotifyService)
	step := planDelegateConductorStep(failingAgent{err: errors.New("agent run abc has unfinished plan steps: Delegate readiness notification to comms agent")}, taskSvc, notifySvc)
	if _, err := step(context.Background(), flow.State{}); err != nil {
		t.Fatalf("planDelegateConductorStep returned %v, want require-notify recovery to run", err)
	}
}

func TestPlanDelegateConductorKeepsUnfinishedTaskFailureActionable(t *testing.T) {
	step := planDelegateConductorStep(failingAgent{err: errors.New("agent run abc has unfinished plan steps: Create Build task")}, new(TaskService), new(NotifyService))
	err := func() error { _, err := step(context.Background(), flow.State{}); return err }()
	if err == nil {
		t.Fatal("planDelegateConductorStep returned nil, want unfinished task error")
	}
	if got := err.Error(); !strings.Contains(got, "Create Build task") {
		t.Fatalf("error = %q, want original unfinished task detail", got)
	}
}

func TestPlanDelegateExecutionRecoversMissingNotifyOnce(t *testing.T) {
	taskSvc := new(TaskService)
	for _, title := range []string{"Design", "Build", "Ship"} {
		var rsp AddResponse
		if err := taskSvc.Add(context.Background(), &AddRequest{Title: title}, &rsp); err != nil {
			t.Fatalf("Add(%q): %v", title, err)
		}
	}
	notifySvc := new(NotifyService)

	recovered := false
	_, err := requireDelegatedNotifyStep(taskSvc, notifySvc, func(ctx context.Context) error {
		recovered = true
		var rsp SendResponse
		return notifySvc.Send(ctx, &SendRequest{To: "owner@acme.com", Message: "The launch plan is ready"}, &rsp)
	})(context.Background(), flow.State{})
	if err != nil {
		t.Fatalf("waitForPlanDelegateExecution returned %v, want recovery success", err)
	}
	if !recovered {
		t.Fatal("missing notify recovery was not invoked")
	}
	if got := notifySvc.count(); got != 1 {
		t.Fatalf("notify count = %d, want 1 after recovery", got)
	}
}

func TestPlanDelegateExecutionWaitsForInFlightNotifyAfterFlowCompletion(t *testing.T) {
	taskSvc := new(TaskService)
	for _, title := range []string{"Design", "Build", "Ship"} {
		var rsp AddResponse
		if err := taskSvc.Add(context.Background(), &AddRequest{Title: title}, &rsp); err != nil {
			t.Fatalf("Add(%q): %v", title, err)
		}
	}
	notifySvc := new(NotifyService)

	go func() {
		time.Sleep(100 * time.Millisecond)
		var rsp SendResponse
		_ = notifySvc.Send(context.Background(), &SendRequest{To: "owner@acme.com", Message: "The launch plan is ready"}, &rsp)
	}()

	recovered := false
	_, err := requireDelegatedNotifyStep(taskSvc, notifySvc, func(ctx context.Context) error {
		recovered = true
		var rsp SendResponse
		return notifySvc.Send(ctx, &SendRequest{To: "owner@acme.com", Message: "The launch plan is ready"}, &rsp)
	})(context.Background(), flow.State{})
	if err != nil {
		t.Fatalf("waitForPlanDelegateExecution returned %v, want in-flight notify success", err)
	}
	if recovered {
		t.Fatal("missing notify recovery ran while delegated notify was still in flight")
	}
	if got := taskSvc.count(); got != 3 {
		t.Fatalf("task count = %d, want 3 after in-flight notify settles", got)
	}
	if got := notifySvc.count(); got != 1 {
		t.Fatalf("notify count = %d, want 1 after in-flight notify settles", got)
	}
	if got := notifySvc.duplicateAttempts(); got != 0 {
		t.Fatalf("duplicate notify attempts = %d, want 0", got)
	}
}

func TestPlanDelegateRecoveryWaitsForRecoveredNotifySideEffect(t *testing.T) {
	taskSvc := new(TaskService)
	for _, title := range []string{"Design", "Build", "Ship"} {
		var rsp AddResponse
		if err := taskSvc.Add(context.Background(), &AddRequest{Title: title}, &rsp); err != nil {
			t.Fatalf("Add(%q): %v", title, err)
		}
	}
	notifySvc := new(NotifyService)

	recovered := false
	_, err := requireDelegatedNotifyStep(taskSvc, notifySvc, func(ctx context.Context) error {
		recovered = true
		go func() {
			time.Sleep(100 * time.Millisecond)
			var rsp SendResponse
			_ = notifySvc.Send(ctx, &SendRequest{To: "owner@acme.com", Message: "The launch plan is ready"}, &rsp)
		}()
		return nil
	})(context.Background(), flow.State{})
	if err != nil {
		t.Fatalf("requireDelegatedNotifyStep returned %v, want delayed recovery success", err)
	}
	if !recovered {
		t.Fatal("missing notify recovery did not run")
	}
	if got := notifySvc.count(); got != 1 {
		t.Fatalf("notify count = %d, want recovered notify side effect", got)
	}
}

func TestPlanDelegateExecutionAcceptsClientTimeoutAfterSideEffects(t *testing.T) {
	taskSvc := new(TaskService)
	for _, title := range []string{"Design", "Build", "Ship"} {
		var rsp AddResponse
		if err := taskSvc.Add(context.Background(), &AddRequest{Title: title}, &rsp); err != nil {
			t.Fatalf("Add(%q): %v", title, err)
		}
	}
	notifySvc := new(NotifyService)
	var rsp SendResponse
	if err := notifySvc.Send(context.Background(), &SendRequest{To: "owner@acme.com", Message: "The launch plan is ready"}, &rsp); err != nil {
		t.Fatalf("Send: %v", err)
	}

	done := make(chan error, 1)
	done <- errors.New(`{"id":"go.micro.client","code":408,"detail":"<nil>","status":"Request Timeout"}`)

	if err := waitForPlanDelegateExecution(done, taskSvc, notifySvc, nil); err != nil {
		t.Fatalf("waitForPlanDelegateExecution returned %v, want completed side effects to satisfy client timeout", err)
	}
}

func TestPlanDelegateExecutionAcceptsApprovalPauseAfterSideEffects(t *testing.T) {
	taskSvc := new(TaskService)
	for _, title := range []string{"Design", "Build", "Ship"} {
		var rsp AddResponse
		if err := taskSvc.Add(context.Background(), &AddRequest{Title: title}, &rsp); err != nil {
			t.Fatalf("Add(%q): %v", title, err)
		}
	}
	notifySvc := new(NotifyService)
	var rsp SendResponse
	if err := notifySvc.Send(context.Background(), &SendRequest{To: "owner@acme.com", Message: "The launch plan is ready"}, &rsp); err != nil {
		t.Fatalf("Send: %v", err)
	}

	done := make(chan error, 1)
	done <- errors.New("agent run abc paused for approval: The comms agent is repeatedly timing out (408 errors) while retrying the launch-readiness notification")

	if err := waitForPlanDelegateExecution(done, taskSvc, notifySvc, nil); err != nil {
		t.Fatalf("waitForPlanDelegateExecution returned %v, want completed side effects to satisfy approval pause", err)
	}
}

func TestPlanDelegateExecutionClassifiesClientTimeoutBeforeSideEffects(t *testing.T) {
	done := make(chan error, 1)
	done <- errors.New(`{"id":"go.micro.client","code":408,"detail":"<nil>","status":"Request Timeout"}`)

	err := waitForPlanDelegateExecution(done, new(TaskService), new(NotifyService), nil)
	if err == nil {
		t.Fatal("waitForPlanDelegateExecution returned nil, want timeout before side effects to fail")
	}
	for _, want := range []string{
		"provider latency/outage during plan-delegate",
		"tasks=0/3 notify=0/1",
		"retry live provider or inspect provider logs",
		"Request Timeout",
	} {
		if got := err.Error(); !strings.Contains(got, want) {
			t.Fatalf("error = %q, want %q", got, want)
		}
	}
}

func TestPlanDelegateExecutionClassifiesPartialClientTimeout(t *testing.T) {
	taskSvc := new(TaskService)
	for _, title := range []string{"Design", "Build", "Ship"} {
		var rsp AddResponse
		if err := taskSvc.Add(context.Background(), &AddRequest{Title: title}, &rsp); err != nil {
			t.Fatalf("Add(%q): %v", title, err)
		}
	}
	done := make(chan error, 1)
	done <- errors.New(`{"id":"go.micro.client","code":408,"detail":"<nil>","status":"Request Timeout"}`)

	err := waitForPlanDelegateExecution(done, taskSvc, new(NotifyService), nil)
	if err == nil {
		t.Fatal("waitForPlanDelegateExecution returned nil, want timeout before notify to fail")
	}
	if got := err.Error(); !strings.Contains(got, "tasks=3/3 notify=0/1") {
		t.Fatalf("error = %q, want partial side-effect counts", got)
	}
}

func TestNotifyServiceSendIsIdempotentForDuplicateDelivery(t *testing.T) {
	svc := new(NotifyService)
	messages := []string{
		"The launch plan is ready",
		"The launch plan is ready.",
		"Launch readiness: the plan is ready!",
	}
	for i, message := range messages {
		var rsp SendResponse
		to := "owner@acme.com"
		if i == len(messages)-1 {
			to = "owner"
		}
		if err := svc.Send(context.Background(), &SendRequest{To: to, Message: message}, &rsp); err != nil {
			t.Fatalf("Send attempt %d: %v", i+1, err)
		}
		if !rsp.Sent {
			t.Fatalf("Send attempt %d reported Sent=false", i+1)
		}
	}
	if got := svc.count(); got != 1 {
		t.Fatalf("notify count = %d, want 1 after duplicate delivery replays", got)
	}
	if got := svc.duplicateAttempts(); got != len(messages)-1 {
		t.Fatalf("duplicate notify attempts = %d, want %d", got, len(messages)-1)
	}
}

func TestNotifyServiceCollapsesProviderReadinessParaphrases(t *testing.T) {
	svc := new(NotifyService)
	requests := []SendRequest{
		{To: "owner@acme.com", Message: "The launch plan is ready"},
		{To: "owner @ acme.com", Message: "Launch plan ready."},
		{To: "owner at acme dot com", Message: "The launch plan is ready."},
		{To: "launch owner", Message: "The launch readiness plan is prepared."},
		{To: "plan owner", Message: "Launch plan is complete!"},
	}
	for i, req := range requests {
		var rsp SendResponse
		if err := svc.Send(context.Background(), &req, &rsp); err != nil {
			t.Fatalf("Send attempt %d: %v", i+1, err)
		}
		if !rsp.Sent {
			t.Fatalf("Send attempt %d reported Sent=false", i+1)
		}
	}
	if got := svc.count(); got != 1 {
		t.Fatalf("notify count = %d, want 1 after provider paraphrase replays", got)
	}
	if got := svc.duplicateAttempts(); got != len(requests)-1 {
		t.Fatalf("duplicate notify attempts = %d, want %d", got, len(requests)-1)
	}
}
