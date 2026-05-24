package tools

import (
	"testing"

	"go-micro.dev/v5/registry"
)

func TestJSONType(t *testing.T) {
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
		if got := jsonType(in); got != want {
			t.Errorf("jsonType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFromRegistry_Empty(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	tools, err := FromRegistry(reg)
	if err != nil {
		t.Fatalf("FromRegistry: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

func TestFromRegistry_DiscoversEndpoints(t *testing.T) {
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

	tools, err := FromRegistry(reg)
	if err != nil {
		t.Fatalf("FromRegistry: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if tool.Name != "users_Users_Get" {
		t.Errorf("safe name = %q, want users_Users_Get", tool.Name)
	}
	if tool.OriginalName != "users.Users.Get" {
		t.Errorf("original = %q", tool.OriginalName)
	}
	if tool.Description != "Fetch a user by ID" {
		t.Errorf("description = %q", tool.Description)
	}

	id, ok := tool.Properties["id"].(map[string]any)
	if !ok {
		t.Fatal("missing id property")
	}
	if id["type"] != "string" {
		t.Errorf("id type = %v", id["type"])
	}
	expand, ok := tool.Properties["expand"].(map[string]any)
	if !ok {
		t.Fatal("missing expand property")
	}
	if expand["type"] != "boolean" {
		t.Errorf("expand type = %v", expand["type"])
	}
}

func TestSet_HandlerResolvesSafeName(t *testing.T) {
	s := New(registry.NewMemoryRegistry())
	s.names.put("users_Users_Get", "users.Users.Get")

	resolved, ok := s.names.get("users_Users_Get")
	if !ok || resolved != "users.Users.Get" {
		t.Errorf("name map lookup = (%q, %v)", resolved, ok)
	}
}

func TestSet_HandlerInvalidName(t *testing.T) {
	s := New(registry.NewMemoryRegistry())
	h := s.Handler(nil)

	// "foo" has no dot, no mapping entry — should error cleanly.
	result, content := h("foo", map[string]any{})
	if result == nil {
		t.Fatal("expected error result")
	}
	if content == "" {
		t.Error("expected non-empty content")
	}
}
