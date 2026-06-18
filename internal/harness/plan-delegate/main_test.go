package main

import (
	"context"
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
	task := service.New(service.Name("task"), service.Registry(reg), service.Client(cl))
	if err := task.Handle(taskSvc); err != nil {
		t.Fatalf("handle task: %v", err)
	}
	go task.Run()

	notifySvc := new(NotifyService)
	notify := service.New(service.Name("notify"), service.Registry(reg), service.Client(cl))
	if err := notify.Handle(notifySvc); err != nil {
		t.Fatalf("handle notify: %v", err)
	}
	go notify.Run()

	// Real comms agent (owns notify), registered so delegate reaches it over RPC.
	comms := agent.New(
		agent.Name("comms"),
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
	task := service.New(service.Name("task"), service.Registry(reg), service.Client(cl))
	if err := task.Handle(taskSvc); err != nil {
		t.Fatalf("handle task: %v", err)
	}
	go task.Run()

	notifySvc := new(NotifyService)
	notify := service.New(service.Name("notify"), service.Registry(reg), service.Client(cl))
	if err := notify.Handle(notifySvc); err != nil {
		t.Fatalf("handle notify: %v", err)
	}
	go notify.Run()

	comms := agent.New(
		agent.Name("comms"),
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
