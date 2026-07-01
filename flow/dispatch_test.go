package flow

import (
	"context"
	"encoding/json"
	"testing"

	"go-micro.dev/v6/client"
	codecbytes "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/store"
)

// fakeClient embeds the default client (so NewRequest works) and
// overrides Call with a test-supplied function.
type fakeClient struct {
	client.Client
	callFn func(req client.Request, rsp interface{}) error
}

func (c *fakeClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	return c.callFn(req, rsp)
}

// A flow with Agent set hands the rendered prompt to that agent's
// Agent.Chat endpoint over RPC and records its reply — it does not run
// its own model.
func TestExecuteDispatchesToAgent(t *testing.T) {
	f := New("welcome", Agent("comms"), Prompt("welcome {{.Data}}"))

	var svc, ep, parentID string
	f.client = &fakeClient{
		Client: client.DefaultClient,
		callFn: func(req client.Request, rsp interface{}) error {
			svc, ep = req.Service(), req.Endpoint()
			reqFrame := req.Body().(*codecbytes.Frame)
			var body map[string]string
			if err := json.Unmarshal(reqFrame.Data, &body); err != nil {
				t.Fatalf("request body: %v", err)
			}
			parentID = body["parent_id"]
			frame := rsp.(*codecbytes.Frame)
			frame.Data = []byte(`{"reply":"welcomed bob","agent":"comms"}`)
			return nil
		},
	}

	if err := f.Execute(context.Background(), "bob"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if svc != "comms" || ep != "Agent.Chat" {
		t.Errorf("dispatched to %s.%s, want comms.Agent.Chat", svc, ep)
	}
	if parentID == "" {
		t.Fatal("dispatch request parent_id is empty")
	}

	results := f.Results()
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Reply != "welcomed bob" {
		t.Errorf("result reply = %q, want %q", results[0].Reply, "welcomed bob")
	}
	if results[0].Prompt != "welcome bob" {
		t.Errorf("rendered prompt = %q, want %q", results[0].Prompt, "welcome bob")
	}
}

// A caller-owned schedule can trigger an agent workflow without a human chat
// prompt and still leave the normal flow run metadata behind for inspection.
func TestScheduledAgentRunHarnessContract(t *testing.T) {
	ctx := context.Background()
	cp := StoreCheckpoint(store.NewMemoryStore(), "scheduled-contract")
	f := New("scheduled-contract",
		Trigger("schedule.daily"),
		WithCheckpoint(cp),
		Steps(Step{Name: "summarize", Run: Dispatch("ops-agent")}),
	)

	var parentID string
	f.client = &fakeClient{
		Client: client.DefaultClient,
		callFn: func(req client.Request, rsp interface{}) error {
			if req.Service() != "ops-agent" || req.Endpoint() != "Agent.Chat" {
				t.Fatalf("dispatched to %s.%s, want ops-agent.Agent.Chat", req.Service(), req.Endpoint())
			}
			reqFrame := req.Body().(*codecbytes.Frame)
			var body map[string]string
			if err := json.Unmarshal(reqFrame.Data, &body); err != nil {
				t.Fatalf("request body: %v", err)
			}
			parentID = body["parent_id"]
			if body["message"] != "run unattended daily ops review" {
				t.Fatalf("message = %q, want scheduled payload", body["message"])
			}
			frame := rsp.(*codecbytes.Frame)
			frame.Data = []byte(`{"reply":"review queued","agent":"ops-agent","parent_id":"` + parentID + `"}`)
			return nil
		},
	}

	if err := Scheduled(f, "run unattended daily ops review").Tick(ctx); err != nil {
		t.Fatalf("scheduled tick: %v", err)
	}
	if parentID == "" {
		t.Fatal("dispatch did not receive the scheduled flow run id as parent_id")
	}

	runs, err := cp.List(ctx)
	if err != nil {
		t.Fatalf("list scheduled runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("got %d runs, want 1", len(runs))
	}
	run := runs[0]
	if run.ID != parentID {
		t.Fatalf("run ID = %q, parent_id = %q", run.ID, parentID)
	}
	if run.Flow != "scheduled-contract" || run.Status != "done" {
		t.Fatalf("run = %+v, want scheduled-contract done", run)
	}
	if got := run.State.String(); got != "review queued" {
		t.Fatalf("run result = %q, want agent reply", got)
	}
	if len(run.Steps) != 1 || run.Steps[0].Name != "summarize" || run.Steps[0].Status != "done" {
		t.Fatalf("steps = %+v, want summarize done", run.Steps)
	}
}
