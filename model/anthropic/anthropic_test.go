package anthropic

import (
	"encoding/json"
	"testing"

	"go-micro.dev/v5/model"
)

func TestProvider_Name(t *testing.T) {
	p := NewProvider(model.Options{})
	if p.Name() != "anthropic" {
		t.Errorf("Expected provider name 'anthropic', got '%s'", p.Name())
	}
}

func TestProvider_DefaultModel(t *testing.T) {
	p := NewProvider(model.Options{})
	if p.DefaultModel() != "claude-sonnet-4-20250514" {
		t.Errorf("Expected default model 'claude-sonnet-4-20250514', got '%s'", p.DefaultModel())
	}
}

func TestProvider_DefaultBaseURL(t *testing.T) {
	p := NewProvider(model.Options{})
	if p.DefaultBaseURL() != "https://api.anthropic.com" {
		t.Errorf("Expected default base URL 'https://api.anthropic.com', got '%s'", p.DefaultBaseURL())
	}
}

func TestProvider_GetAPIEndpoint(t *testing.T) {
	p := NewProvider(model.Options{})
	baseURL := "https://api.anthropic.com"
	endpoint := p.GetAPIEndpoint(baseURL)
	expected := "https://api.anthropic.com/v1/messages"
	if endpoint != expected {
		t.Errorf("Expected endpoint '%s', got '%s'", expected, endpoint)
	}
}

func TestProvider_SetAuthHeaders(t *testing.T) {
	p := NewProvider(model.Options{})
	headers := make(map[string]string)
	apiKey := "test-api-key"
	p.SetAuthHeaders(headers, apiKey)

	if headers["x-api-key"] != apiKey {
		t.Errorf("Expected x-api-key header to be '%s', got '%s'", apiKey, headers["x-api-key"])
	}
	if headers["anthropic-version"] != "2023-06-01" {
		t.Errorf("Expected anthropic-version header to be '2023-06-01', got '%s'", headers["anthropic-version"])
	}
}

func TestProvider_BuildRequest(t *testing.T) {
	p := NewProvider(model.Options{})
	
	tools := []model.Tool{
		{
			Name:        "test_tool",
			Description: "A test tool",
			Properties: map[string]any{
				"param1": map[string]any{
					"type":        "string",
					"description": "A test parameter",
				},
			},
		},
	}

	body, err := p.BuildRequest("test prompt", "system prompt", tools, nil)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}

	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if req["model"] != p.DefaultModel() {
		t.Errorf("Expected model '%s', got '%v'", p.DefaultModel(), req["model"])
	}
	if req["system"] != "system prompt" {
		t.Errorf("Expected system prompt 'system prompt', got '%v'", req["system"])
	}
	if req["max_tokens"] != float64(4096) {
		t.Errorf("Expected max_tokens 4096, got %v", req["max_tokens"])
	}

	messages, ok := req["messages"].([]any)
	if !ok || len(messages) == 0 {
		t.Fatal("Expected messages array")
	}

	reqTools, ok := req["tools"].([]any)
	if !ok || len(reqTools) != 1 {
		t.Fatal("Expected tools array with 1 tool")
	}
}

func TestProvider_ParseResponse(t *testing.T) {
	p := NewProvider(model.Options{})

	// Test response with text only
	responseJSON := `{
		"content": [
			{"type": "text", "text": "Hello, world!"}
		],
		"stop_reason": "end_turn"
	}`

	resp, err := p.ParseResponse([]byte(responseJSON))
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if resp.Reply != "Hello, world!" {
		t.Errorf("Expected reply 'Hello, world!', got '%s'", resp.Reply)
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("Expected 0 tool calls, got %d", len(resp.ToolCalls))
	}

	// Test response with tool use
	responseWithToolJSON := `{
		"content": [
			{"type": "text", "text": "I'll call a tool"},
			{"type": "tool_use", "id": "tool_123", "name": "test_tool", "input": {"param1": "value1"}}
		],
		"stop_reason": "tool_use"
	}`

	resp2, err := p.ParseResponse([]byte(responseWithToolJSON))
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if resp2.Reply != "I'll call a tool" {
		t.Errorf("Expected reply 'I'll call a tool', got '%s'", resp2.Reply)
	}
	if len(resp2.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp2.ToolCalls))
	}
	if resp2.ToolCalls[0].ID != "tool_123" {
		t.Errorf("Expected tool call ID 'tool_123', got '%s'", resp2.ToolCalls[0].ID)
	}
	if resp2.ToolCalls[0].Name != "test_tool" {
		t.Errorf("Expected tool call name 'test_tool', got '%s'", resp2.ToolCalls[0].Name)
	}
}

func TestProvider_ParseFollowUpResponse(t *testing.T) {
	p := NewProvider(model.Options{})

	responseJSON := `{
		"content": [
			{"type": "text", "text": "The result is: 42"}
		]
	}`

	answer, err := p.ParseFollowUpResponse([]byte(responseJSON))
	if err != nil {
		t.Fatalf("ParseFollowUpResponse failed: %v", err)
	}

	if answer != "The result is: 42" {
		t.Errorf("Expected answer 'The result is: 42', got '%s'", answer)
	}
}
