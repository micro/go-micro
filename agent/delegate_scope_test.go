package agent

import (
	"context"
	"strings"
	"testing"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
)

func TestDelegateServiceScope(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	for _, name := range []string{"allowed", "private"} {
		if err := reg.Register(&registry.Service{Name: name, Version: "1", Nodes: []*registry.Node{{Id: name, Address: "127.0.0.1:1"}}, Endpoints: []*registry.Endpoint{{Name: "Service.Read"}}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name  string
		opts  []Option
		count int
	}{
		{"unrestricted", nil, 2},
		{"restricted", []Option{Services("allowed")}, 1},
		{"explicit empty", []Option{Services()}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			fakeGen = func(ctx context.Context, opts model.Options, req *model.Request) (*model.Response, error) {
				calls++
				if len(req.Tools) != tc.count {
					t.Errorf("sub-agent tools = %v, want %d", req.Tools, tc.count)
				}
				if tc.count == 1 && !strings.HasPrefix(req.Tools[0].OriginalName, "allowed.") {
					t.Errorf("unexpected tool: %v", req.Tools[0])
				}
				return &model.Response{Reply: "done"}, nil
			}
			defer func() { fakeGen = nil }()
			a := newTestAgent(append([]Option{Name("parent"), WithRegistry(reg)}, tc.opts...)...)
			res := a.handleDelegate(context.Background(), model.ToolCall{ID: "delegate", Input: map[string]any{"task": "read"}})
			if calls != 1 {
				t.Fatalf("model calls = %d; result: %+v", calls, res)
			}
		})
	}
}

func TestDelegateRejectsOutOfScopeTarget(t *testing.T) {
	a := newTestAgent(Services("allowed"))
	for _, target := range []string{"private"} {
		res := a.handleDelegate(context.Background(), model.ToolCall{ID: target, Input: map[string]any{"task": "read", "to": target}})
		if !strings.Contains(res.Content, "outside the agent's service scope") {
			t.Fatalf("target %q was not refused: %+v", target, res)
		}
	}
}
