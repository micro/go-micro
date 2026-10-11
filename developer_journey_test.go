package micro_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	micro "go-micro.dev/v6"
	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

type JourneyRequest struct{}
type JourneyResponse struct {
	Notes []string `json:"notes"`
}
type JourneyNotes struct{ calls atomic.Int32 }

func (n *JourneyNotes) List(_ context.Context, _ *JourneyRequest, rsp *JourneyResponse) error {
	n.calls.Add(1)
	rsp.Notes = []string{"Review the release"}
	return nil
}

// Exercise real RPC and provider HTTP adapters without model credentials or mDNS.
func TestDeveloperJourney(t *testing.T) {
	ctx := context.Background()
	reg := registry.NewMemoryRegistry()
	br := broker.NewMemoryBroker()
	svc := micro.NewService("notes", micro.Address("127.0.0.1:0"), micro.Broker(br), micro.Registry(reg))
	notes := &JourneyNotes{}
	if err := svc.Handle(notes); err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	defer svc.Stop()
	var result JourneyResponse
	if err := svc.Client().Call(ctx, svc.Client().NewRequest("notes", "JourneyNotes.List", &JourneyRequest{}), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Notes) != 1 {
		t.Fatalf("RPC result: %+v", result)
	}

	for _, provider := range []string{"openai", "mistral"} {
		t.Run(provider, func(t *testing.T) {
			before := notes.calls.Load()
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("unexpected provider request: %s", r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var req struct {
					Messages []struct {
						Role    string
						Content any
					}
					Tools []struct{ Function struct{ Name string } }
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				for _, m := range req.Messages {
					if m.Role == "tool" {
						if !strings.Contains(m.Content.(string), "Review the release") {
							t.Error("missing service result")
						}
						_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Review the release"}}]}`))
						return
					}
				}
				for _, tool := range req.Tools {
					if strings.Contains(tool.Function.Name, "JourneyNotes") {
						_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{
							"role": "assistant", "content": "", "tool_calls": []any{map[string]any{"id": "call00001", "type": "function", "function": map[string]any{"name": tool.Function.Name, "arguments": "{}"}}},
						}}}})
						return
					}
				}
				t.Error("service was not exposed as a tool")
				w.WriteHeader(http.StatusBadRequest)
			}))
			defer api.Close()
			a := micro.NewAgent("assistant", micro.AgentProvider(provider), micro.AgentAPIKey("test-key"), micro.AgentBaseURL(api.URL),
				micro.AgentServices("notes"), agent.WithRegistry(reg), agent.WithClient(svc.Client()), agent.WithStore(store.NewMemoryStore()))
			reply, err := a.Ask(ctx, "What notes do I have?")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(reply.Reply, "Review the release") || notes.calls.Load() != before+1 {
				t.Fatalf("agent did not call service: %+v", reply)
			}
		})
	}

	dir := t.TempDir()
	newStore := func() store.Store { return store.NewFileStore(store.DirOption(dir)) }
	st := newStore()
	fail := true
	makeFlow := func(st store.Store) *flow.Flow {
		f := flow.New("read-notes", flow.WithCheckpoint(flow.StoreCheckpoint(st, "read-notes")), flow.Steps(
			flow.Step{Name: "list", Run: flow.Call("notes", "JourneyNotes.List")},
			flow.Step{Name: "save", Run: func(_ context.Context, in flow.State) (flow.State, error) {
				if fail {
					return in, errors.New("destination unavailable")
				}
				if !strings.Contains(in.String(), "Review the release") {
					t.Error("lost service response")
				}
				return in, nil
			}},
		))
		if err := f.Register(reg, br, svc.Client()); err != nil {
			t.Fatal(err)
		}
		return f
	}
	f := makeFlow(st)
	if err := f.Execute(ctx, "{}"); err == nil {
		t.Fatal("expected interrupted workflow")
	}
	before := notes.calls.Load()
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st = newStore()
	defer st.Close()
	fail = false
	f = makeFlow(st)
	if _, err := f.ResumePending(ctx); err != nil {
		t.Fatal(err)
	}
	if notes.calls.Load() != before {
		t.Fatal("resume repeated completed service call")
	}
	pending, err := f.Pending(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("unfinished workflow: %v %v", pending, err)
	}
}
