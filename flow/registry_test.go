package flow

import (
	"context"
	"testing"

	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/registry"
)

// A trigger-bound flow announces itself in the registry as type=flow
// while running, and deregisters on Stop — liveness, like a service.
func TestFlowRegistersAndDeregisters(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	br := broker.NewMemoryBroker()
	if err := br.Connect(); err != nil {
		t.Fatalf("broker connect: %v", err)
	}

	f := New("onboard",
		Trigger("events.user.created"),
		Steps(appendStep("a")),
	)
	if err := f.Register(reg, br, client.DefaultClient); err != nil {
		t.Fatalf("Register: %v", err)
	}

	svcs, err := reg.GetService("onboard")
	if err != nil || len(svcs) == 0 {
		t.Fatalf("flow not registered: %v", err)
	}
	if svcs[0].Metadata["type"] != "flow" {
		t.Errorf("registry metadata type = %q, want flow", svcs[0].Metadata["type"])
	}
	if svcs[0].Metadata["trigger"] != "events.user.created" {
		t.Errorf("registry metadata trigger = %q", svcs[0].Metadata["trigger"])
	}

	if err := f.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if svcs, _ := reg.GetService("onboard"); len(svcs) != 0 {
		t.Errorf("flow should be deregistered after Stop, got %d services", len(svcs))
	}
}

// A flow without a trigger is not a running listener and isn't registered.
func TestFlowWithoutTriggerNotRegistered(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	br := broker.NewMemoryBroker()
	_ = br.Connect()

	f := New("oneshot", Steps(appendStep("a")))
	if err := f.Register(reg, br, client.DefaultClient); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if svcs, _ := reg.GetService("oneshot"); len(svcs) != 0 {
		t.Errorf("triggerless flow should not register, got %d", len(svcs))
	}
	// It still runs on demand.
	if err := f.Execute(context.Background(), ""); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}
