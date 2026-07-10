package main

import (
	"context"
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
	ws := service.New(service.Name("workspace"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	if err := ws.Handle(wsSvc); err != nil {
		t.Fatalf("handle workspace: %v", err)
	}
	go ws.Run()

	ntSvc := new(NotifyService)
	nt := service.New(service.Name("notify"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	if err := nt.Handle(ntSvc); err != nil {
		t.Fatalf("handle notify: %v", err)
	}
	go nt.Run()

	onboarder := agent.New(
		agent.Name("onboarder"),
		agent.Address("127.0.0.1:0"),
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

func TestWaitForOnboardingSideEffectsFailsWhenMissing(t *testing.T) {
	wsSvc := new(WorkspaceService)
	ntSvc := new(NotifyService)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	err := waitForOnboardingSideEffects(ctx, wsSvc, ntSvc)
	if err == nil {
		t.Fatal("waitForOnboardingSideEffects returned nil, want missing side effects error")
	}
	if got := err.Error(); !strings.Contains(got, "workspaces=0/1") || !strings.Contains(got, "notifications=0/1") {
		t.Fatalf("waitForOnboardingSideEffects error %q does not report missing side effects", got)
	}
}

func TestWaitForOnboardingSideEffectsPassesWhenComplete(t *testing.T) {
	wsSvc := new(WorkspaceService)
	ntSvc := new(NotifyService)

	if err := wsSvc.Create(context.Background(), &CreateRequest{Owner: "alice@acme.com"}, &CreateResponse{}); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := ntSvc.Send(context.Background(), &SendRequest{To: "alice@acme.com", Message: "Welcome"}, &SendResponse{}); err != nil {
		t.Fatalf("send notification: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := waitForOnboardingSideEffects(ctx, wsSvc, ntSvc); err != nil {
		t.Fatalf("waitForOnboardingSideEffects returned %v, want nil", err)
	}
}

func TestWorkspaceCreateSuppressesDuplicateOwner(t *testing.T) {
	wsSvc := new(WorkspaceService)
	first := new(CreateResponse)
	if err := wsSvc.Create(context.Background(), &CreateRequest{Owner: "alice@acme.com"}, first); err != nil {
		t.Fatalf("create first workspace: %v", err)
	}
	second := new(CreateResponse)
	if err := wsSvc.Create(context.Background(), &CreateRequest{Owner: "alice@acme.com"}, second); err != nil {
		t.Fatalf("create duplicate workspace: %v", err)
	}

	if got := wsSvc.count(); got != 1 {
		t.Fatalf("workspace creations = %d, want 1 after duplicate owner replay", got)
	}
	if first.Workspace == nil || second.Workspace == nil || second.Workspace.ID != first.Workspace.ID {
		t.Fatalf("duplicate create returned workspace %#v, want original %#v", second.Workspace, first.Workspace)
	}
}

func TestNotifySendSuppressesDuplicateMessage(t *testing.T) {
	ntSvc := new(NotifyService)
	req := &SendRequest{To: "alice@acme.com", Message: "Welcome — your workspace is ready."}
	if err := ntSvc.Send(context.Background(), req, &SendResponse{}); err != nil {
		t.Fatalf("send first notification: %v", err)
	}
	if err := ntSvc.Send(context.Background(), req, &SendResponse{}); err != nil {
		t.Fatalf("send duplicate notification: %v", err)
	}

	if got := ntSvc.count(); got != 1 {
		t.Fatalf("notifications sent = %d, want 1 after duplicate message replay", got)
	}
}
