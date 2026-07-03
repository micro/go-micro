package main

import (
	"context"
	"errors"
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
	go task.Run()

	notifySvc := new(NotifyService)
	notify := service.New(service.Name("notify"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	if err := notify.Handle(notifySvc); err != nil {
		t.Fatalf("handle notify: %v", err)
	}
	go notify.Run()

	// Real comms agent (owns notify), registered so delegate reaches it over RPC.
	comms := agent.New(
		agent.Name("comms"),
		agent.Address("127.0.0.1:0"),
		agent.Services("notify"),
		agent.Prompt("You handle outbound notifications."),
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
	go task.Run()

	notifySvc := new(NotifyService)
	notify := service.New(service.Name("notify"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	if err := notify.Handle(notifySvc); err != nil {
		t.Fatalf("handle notify: %v", err)
	}
	go notify.Run()

	comms := agent.New(
		agent.Name("comms"),
		agent.Address("127.0.0.1:0"),
		agent.Services("notify"),
		agent.Prompt("You handle outbound notifications."),
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

func TestPlanDelegateExecutionReportsDuplicateNotifyBeforeTimeout(t *testing.T) {
	notifySvc := new(NotifyService)
	for i := 0; i < 2; i++ {
		var rsp SendResponse
		if err := notifySvc.Send(context.Background(), &SendRequest{To: "owner@acme.com", Message: "The launch plan is ready"}, &rsp); err != nil {
			t.Fatalf("Send attempt %d: %v", i+1, err)
		}
	}

	done := make(chan error)
	errCh := make(chan error, 1)
	go func() { errCh <- waitForPlanDelegateExecution(done, new(TaskService), notifySvc, nil) }()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("waitForPlanDelegateExecution returned nil, want duplicate notify error")
		}
		if got := err.Error(); !strings.Contains(got, "duplicate notify attempts") {
			t.Fatalf("error = %q, want duplicate notify attempts", got)
		}
	case <-time.After(time.Second):
		t.Fatal("waitForPlanDelegateExecution did not report duplicate notify before timeout")
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

func TestPlanDelegateExecutionRecoversMissingNotifyOnce(t *testing.T) {
	taskSvc := new(TaskService)
	for _, title := range []string{"Design", "Build", "Ship"} {
		var rsp AddResponse
		if err := taskSvc.Add(context.Background(), &AddRequest{Title: title}, &rsp); err != nil {
			t.Fatalf("Add(%q): %v", title, err)
		}
	}
	notifySvc := new(NotifyService)
	done := make(chan error, 1)
	done <- nil

	recovered := false
	err := waitForPlanDelegateExecution(done, taskSvc, notifySvc, func(ctx context.Context) error {
		recovered = true
		var rsp SendResponse
		return notifySvc.Send(ctx, &SendRequest{To: "owner@acme.com", Message: "The launch plan is ready"}, &rsp)
	})
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

func TestPlanDelegateExecutionRejectsClientTimeoutBeforeSideEffects(t *testing.T) {
	done := make(chan error, 1)
	done <- errors.New(`{"id":"go.micro.client","code":408,"detail":"<nil>","status":"Request Timeout"}`)

	err := waitForPlanDelegateExecution(done, new(TaskService), new(NotifyService), nil)
	if err == nil {
		t.Fatal("waitForPlanDelegateExecution returned nil, want timeout before side effects to fail")
	}
	if got := err.Error(); !strings.Contains(got, "tasks=0 notify=0") {
		t.Fatalf("error = %q, want side-effect counts", got)
	}
}

func TestNotifyServiceSendIsIdempotentForDuplicateDelivery(t *testing.T) {
	svc := new(NotifyService)
	for i := 0; i < 3; i++ {
		var rsp SendResponse
		if err := svc.Send(context.Background(), &SendRequest{To: "owner@acme.com", Message: "The launch plan is ready"}, &rsp); err != nil {
			t.Fatalf("Send attempt %d: %v", i+1, err)
		}
		if !rsp.Sent {
			t.Fatalf("Send attempt %d reported Sent=false", i+1)
		}
	}
	if got := svc.count(); got != 1 {
		t.Fatalf("notify count = %d, want 1 after duplicate delivery replays", got)
	}
}
