package flow

import (
	"context"
	"testing"

	"go-micro.dev/v6/client"
	codecbytes "go-micro.dev/v6/codec/bytes"
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

	var svc, ep string
	f.client = &fakeClient{
		Client: client.DefaultClient,
		callFn: func(req client.Request, rsp interface{}) error {
			svc, ep = req.Service(), req.Endpoint()
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
