package openai

import (
	"encoding/json"
	"testing"

	"go-micro.dev/v5/model"
)

func TestProvider_Name(t *testing.T) {
	p := NewProvider(model.Options{})
	if p.Name() != "openai" {
		t.Errorf("Expected provider name 'openai', got '%s'", p.Name())
	}
}

func TestProvider_DefaultModel(t *testing.T) {
	p := NewProvider(model.Options{})
	if p.DefaultModel() != "gpt-4o" {
		t.Errorf("Expected default model 'gpt-4o', got '%s'", p.DefaultModel())
	}
}

func TestProvider_DefaultBaseURL(t *testing.T) {
	p := NewProvider(model.Options{})
	if p.DefaultBaseURL() != "https://api.openai.com" {
		t.Errorf("Expected default base URL 'https://api.openai.com', got '%s'", p.DefaultBaseURL())
	}
}

func TestProvider_GetAPIEndpoint(t *testing.T) {
	p := NewProvider(model.Options{})
	baseURL := "https://api.openai.com"
	endpoint := p.GetAPIEndpoint(baseURL)
	expected := "https://api.openai.com/v1/chat/completions"
	if endpoint != expected {
		t.Errorf("Expected endpoint '%s', got '%s'", expected, endpoint)
	}
}

func TestProvider_SetAuthHeaders(t *testing.T) {
	p := NewProvider(model.Options{})
	headers := make(map[string]string)
	apiKey := "test-api-key"
	p.SetAuthHeaders(headers, apiKey)

	expected := "Bearer test-api-key"
	if headers["Authorization"] != expected {
		t.Errorf("Expected Authorization header to be '%s', got '%s'", expected, headers["Authorization"])
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

	messages, ok := req["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatal("Expected messages array with 2 messages")
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
		"choices": [
			{
				"message": {
					"content": "Hello, world!",
					"tool_calls": []
				}
			}
		]
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

	// Test response with tool calls
	responseWithToolJSON := `{
		"choices": [
			{
				"message": {
					"content": "I'll call a tool",
					"tool_calls": [
						{
							"id": "call_123",
							"function": {
								"name": "test_tool",
								"arguments": "{\"param1\": \"value1\"}"
							}
						}
					]
				}
			}
		]
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
	if resp2.ToolCalls[0].ID != "call_123" {
		t.Errorf("Expected tool call ID 'call_123', got '%s'", resp2.ToolCalls[0].ID)
	}
	if resp2.ToolCalls[0].Name != "test_tool" {
		t.Errorf("Expected tool call name 'test_tool', got '%s'", resp2.ToolCalls[0].Name)
	}
}

func TestProvider_ParseFollowUpResponse(t *testing.T) {
	p := NewProvider(model.Options{})

	responseJSON := `{
		"choices": [
			{
				"message": {
					"content": "The result is: 42"
				}
			}
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
