package ai

import (
	"context"
	"testing"

	"go-micro.dev/v6/registry"
)

func TestToolJSONType(t *testing.T) {
	cases := map[string]string{
		"string":  "string",
		"int":     "integer",
		"int64":   "integer",
		"float64": "number",
		"bool":    "boolean",
		"User":    "object",
		"":        "object",
	}
	for in, want := range cases {
		if got := toolJSONType(in); got != want {
			t.Errorf("toolJSONType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDiscoverTools_Empty(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	tools, err := DiscoverTools(reg)
	if err != nil {
		t.Fatalf("DiscoverTools: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

func TestDiscoverTools_DiscoversEndpoints(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	svc := &registry.Service{
		Name:    "users",
		Version: "1.0.0",
		Nodes: []*registry.Node{
			{Id: "users-1", Address: "127.0.0.1:9000"},
		},
		Endpoints: []*registry.Endpoint{
			{
				Name: "Users.Get",
				Metadata: map[string]string{
					"description": "Fetch a user by ID",
				},
				Request: &registry.Value{
					Name: "GetRequest",
					Type: "GetRequest",
					Values: []*registry.Value{
						{Name: "id", Type: "string"},
						{Name: "expand", Type: "bool"},
					},
				},
			},
		},
	}
	if err := reg.Register(svc); err != nil {
		t.Fatalf("Register: %v", err)
	}

	tools, err := DiscoverTools(reg)
	if err != nil {
		t.Fatalf("DiscoverTools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if tool.Name != "users_Users_Get" {
		t.Errorf("safe name = %q", tool.Name)
	}
	if tool.OriginalName != "users.Users.Get" {
		t.Errorf("original = %q", tool.OriginalName)
	}
	if tool.Description != "Fetch a user by ID" {
		t.Errorf("description = %q", tool.Description)
	}
}

func TestTools_HandlerResolvesSafeName(t *testing.T) {
	tools := NewTools(registry.NewMemoryRegistry())
	tools.names.put("users_Users_Get", "users.Users.Get")

	resolved, ok := tools.names.get("users_Users_Get")
	if !ok || resolved != "users.Users.Get" {
		t.Errorf("name map lookup = (%q, %v)", resolved, ok)
	}
}

func TestTools_HandlerInvalidName(t *testing.T) {
	tools := NewTools(registry.NewMemoryRegistry())
	h := tools.Handler()

	res := h(context.Background(), ToolCall{Name: "foo", Input: map[string]any{}})
	if res.Value == nil {
		t.Fatal("expected error result")
	}
	if res.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestWithTools(t *testing.T) {
	tools := NewTools(registry.NewMemoryRegistry())
	opts := NewOptions(WithTools(tools))
	if opts.ToolHandler == nil {
		t.Error("WithTools did not set a ToolHandler")
	}
}

// Discovery order is deterministic. The registry iterates a map, so without
// an explicit sort the tool list is shuffled on every call — which silently
// defeats provider prompt caching: Anthropic cache_control and Gemini
// implicit caching both key on a byte-identical prefix, and the tool
// catalogue is the bulk of that prefix.
func TestDiscoverOrderIsDeterministic(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	for _, name := range []string{"zulu", "alpha", "mike", "bravo", "yankee"} {
		if err := reg.Register(&registry.Service{
			Name: name,
			Endpoints: []*registry.Endpoint{
				{Name: "Svc.Do"},
				{Name: "Svc.List"},
			},
			Nodes: []*registry.Node{{Id: name + "-1", Address: "127.0.0.1:0"}},
		}); err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
	}

	tools := NewTools(reg)
	first, err := tools.Discover()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(first) != 10 {
		t.Fatalf("discovered %d tools, want 10", len(first))
	}
	for i := 1; i < len(first); i++ {
		if first[i-1].Name > first[i].Name {
			t.Fatalf("tools not sorted: %q before %q", first[i-1].Name, first[i].Name)
		}
	}
	// Map iteration order varies run to run; several rounds catch a shuffle.
	for round := 0; round < 5; round++ {
		again, err := tools.Discover()
		if err != nil {
			t.Fatalf("discover round %d: %v", round, err)
		}
		for i := range first {
			if again[i].Name != first[i].Name {
				t.Fatalf("round %d: order changed at %d: %q != %q", round, i, again[i].Name, first[i].Name)
			}
		}
	}
}
