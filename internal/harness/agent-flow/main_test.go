package main

import (
	"testing"
	"time"

	"go-micro.dev/v5/agent"
	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/flow"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/selector"
	"go-micro.dev/v5/service"
	"go-micro.dev/v5/store"
)

// TestEventTriggersAgentNoPrompt proves "the event is the prompt": a
// broker event drives a Flow that hands off to a registered agent, which
// reasons and acts through its services — workspace created, welcome
// sent — with no human prompt anywhere. Real services, registry, RPC,
// broker, agent loop, store; only the LLM is mocked. No mDNS, no sleeps
// beyond polling for the asynchronous side effect.
func TestEventTriggersAgentNoPrompt(t *testing.T) {
	ai.Register("mock", newMock)

	reg := registry.NewMemoryRegistry()
	br := broker.NewMemoryBroker()
	if err := br.Connect(); err != nil {
		t.Fatalf("broker connect: %v", err)
	}
	cl := client.NewClient(
		client.Registry(reg),
		client.Selector(selector.NewSelector(selector.Registry(reg))),
	)
	mem := store.NewMemoryStore()

	wsSvc := new(WorkspaceService)
	ws := service.New(service.Name("workspace"), service.Registry(reg), service.Client(cl))
	if err := ws.Handle(wsSvc); err != nil {
		t.Fatalf("handle workspace: %v", err)
	}
	go ws.Run()

	ntSvc := new(NotifyService)
	nt := service.New(service.Name("notify"), service.Registry(reg), service.Client(cl))
	if err := nt.Handle(ntSvc); err != nil {
		t.Fatalf("handle notify: %v", err)
	}
	go nt.Run()

	onboarder := agent.New(
		agent.Name("onboarder"),
		agent.Services("workspace", "notify"),
		agent.Prompt("You onboard new users. Create their workspace and send a welcome message."),
		agent.Provider("mock"),
		agent.WithRegistry(reg), agent.WithClient(cl), agent.WithStore(mem),
	)
	go onboarder.Run()
	defer onboarder.Stop()

	waitFor(reg, "workspace")
	waitFor(reg, "notify")
	waitFor(reg, "onboarder")

	f := flow.New("onboard",
		flow.Trigger("events.user.created"),
		flow.Agent("onboarder"),
		flow.Prompt("A new user signed up: {{.Data}}. Get them set up."),
	)
	if err := f.Register(reg, br, cl); err != nil {
		t.Fatalf("flow register: %v", err)
	}

	// The event — nobody typed a prompt.
	if err := br.Publish("events.user.created", &broker.Message{
		Body: []byte(`{"email":"alice@acme.com"}`),
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Wait for the agent to act (delivery is asynchronous).
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if wsSvc.count() >= 1 && ntSvc.count() >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if got := wsSvc.count(); got != 1 {
		t.Errorf("workspace created %d times, want 1", got)
	}
	if got := ntSvc.count(); got != 1 {
		t.Errorf("notify sent %d times, want 1 (event->flow->agent chain broken)", got)
	}
	if rs := f.Results(); len(rs) == 0 || rs[len(rs)-1].Reply == "" {
		t.Errorf("flow recorded no result for the event")
	}
}
