package atlascloud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-micro.dev/v6/ai"
)

func TestProvider_String(t *testing.T) {
	p := NewProvider()
	if p.String() != "atlascloud" {
		t.Errorf("Expected provider name 'atlascloud', got '%s'", p.String())
	}
}

func TestProvider_Init(t *testing.T) {
	p := NewProvider()

	err := p.Init(
		ai.WithModel("test-model"),
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL("https://test.com"),
	)

	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	opts := p.Options()
	if opts.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", opts.Model)
	}
	if opts.APIKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", opts.APIKey)
	}
	if opts.BaseURL != "https://test.com" {
		t.Errorf("Expected base URL 'https://test.com', got '%s'", opts.BaseURL)
	}
}

func TestProvider_Options(t *testing.T) {
	p := NewProvider(
		ai.WithModel("custom-model"),
		ai.WithAPIKey("my-key"),
	)

	opts := p.Options()
	if opts.Model != "custom-model" {
		t.Errorf("Expected model 'custom-model', got '%s'", opts.Model)
	}
	if opts.APIKey != "my-key" {
		t.Errorf("Expected API key 'my-key', got '%s'", opts.APIKey)
	}
}

func TestProvider_Defaults(t *testing.T) {
	p := NewProvider()

	opts := p.Options()
	if opts.Model != "deepseek-ai/DeepSeek-V3-0324" {
		t.Errorf("Expected default model 'deepseek-ai/DeepSeek-V3-0324', got '%s'", opts.Model)
	}
	if opts.BaseURL != "https://api.atlascloud.ai" {
		t.Errorf("Expected default base URL 'https://api.atlascloud.ai', got '%s'", opts.BaseURL)
	}
}

func TestProvider_Generate_NoAPIKey(t *testing.T) {
	p := NewProvider()

	req := &ai.Request{
		Prompt:       "Hello",
		SystemPrompt: "You are helpful",
	}

	_, err := p.Generate(context.Background(), req)
	if err == nil {
		t.Error("Expected error when API key is missing, got nil")
	}
}

func TestProvider_Stream(t *testing.T) {
	var sawStream, sawIncludeUsage bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s, want /v1/chat/completions", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		sawStream, _ = body["stream"].(bool)
		if so, ok := body["stream_options"].(map[string]any); ok {
			sawIncludeUsage, _ = so["include_usage"].(bool)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":2,\"total_tokens\":9}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL))
	stream, err := p.Stream(context.Background(), &ai.Request{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	defer stream.Close()
	if !sawStream {
		t.Fatal("stream request did not set stream=true")
	}
	if !sawIncludeUsage {
		t.Fatal("stream request did not set stream_options.include_usage=true")
	}

	first, err := stream.Recv()
	if err != nil || first.Reply != "hel" {
		t.Fatalf("first chunk = %#v, %v; want hel", first, err)
	}
	second, err := stream.Recv()
	if err != nil || second.Reply != "lo" {
		t.Fatalf("second chunk = %#v, %v; want lo", second, err)
	}
	usage, err := stream.Recv()
	if err != nil {
		t.Fatalf("usage chunk error: %v", err)
	}
	if usage.Usage.TotalTokens != 9 || usage.Usage.InputTokens != 7 || usage.Usage.OutputTokens != 2 {
		t.Fatalf("usage = %#v; want input=7 output=2 total=9", usage.Usage)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("final error = %v, want EOF", err)
	}
}

func TestProvider_StreamWithToolsFallsBack(t *testing.T) {
	p := NewProvider(ai.WithAPIKey("test-key"))
	_, err := p.Stream(context.Background(), &ai.Request{
		Prompt: "call a tool",
		Tools: []ai.Tool{{
			Name:        "fallback_echo",
			Description: "echo fallback marker",
			Properties:  map[string]any{"value": map[string]any{"type": "string"}},
		}},
	})
	if !errors.Is(err, ai.ErrStreamingUnsupported) {
		t.Fatalf("Stream with tools error = %v, want ErrStreamingUnsupported", err)
	}
}

func TestProvider_GenerateToolCallEmptyFollowUpUsesToolResult(t *testing.T) {
	var calls int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s, want /v1/chat/completions", r.URL.Path)
		}
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch calls {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-1","function":{"name":"conformance_echo","arguments":"{\"value\":\"agent-conformance\"}"}}]}}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
		default:
			t.Fatalf("unexpected API call %d", calls)
		}
	}))
	defer ts.Close()

	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithToolHandler(func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
			if call.Name != "conformance_echo" {
				t.Fatalf("tool name = %q, want conformance_echo", call.Name)
			}
			return ai.ToolResult{ID: call.ID, Content: `{"marker":"agent-conformance-ok"}`}
		}),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "call a tool",
		Tools: []ai.Tool{{
			Name:        "conformance_echo",
			Description: "echo conformance marker",
			Properties:  map[string]any{"value": map[string]any{"type": "string"}},
		}},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("API calls = %d, want 2", calls)
	}
	if resp.Answer != `{"marker":"agent-conformance-ok"}` {
		t.Fatalf("Answer = %q, want tool result fallback", resp.Answer)
	}
}

func TestProvider_GenerateMinimaxToolRequests(t *testing.T) {
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(bodies) {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-1","function":{"name":"conformance_echo","arguments":"{\"value\":\"agent-conformance\"}"}}]}}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"done"}}]}`))
		default:
			t.Fatalf("unexpected API call %d", len(bodies))
		}
	}))
	defer ts.Close()

	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithModel("minimaxai/minimax-m3"),
		ai.WithToolHandler(func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
			return ai.ToolResult{ID: call.ID, Content: `{"marker":"agent-conformance-ok"}`}
		}),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		SystemPrompt: "You are helpful.",
		Prompt:       "call a tool",
		Tools: []ai.Tool{{
			Name:        "conformance_echo",
			Description: "echo conformance marker",
			Properties:  map[string]any{"value": map[string]any{"type": "string"}},
		}},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.Answer != "done" {
		t.Fatalf("Answer = %q, want done", resp.Answer)
	}
	if len(bodies) != 2 {
		t.Fatalf("captured requests = %d, want 2", len(bodies))
	}
	if got := bodies[0]["model"]; got != "minimaxai/minimax-m3" {
		t.Fatalf("initial model = %v", got)
	}
	tools, ok := bodies[0]["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("initial tools = %#v, want one tool", bodies[0]["tools"])
	}
	tool := tools[0].(map[string]any)
	if tool["type"] != "function" {
		t.Fatalf("tool type = %v, want function", tool["type"])
	}
	fn := tool["function"].(map[string]any)
	if fn["name"] != "conformance_echo" {
		t.Fatalf("tool function name = %v", fn["name"])
	}
	params := fn["parameters"].(map[string]any)
	if params["type"] != "object" {
		t.Fatalf("parameters type = %v, want object", params["type"])
	}

	followUpMessages := bodies[1]["messages"].([]any)
	if len(followUpMessages) != 4 {
		t.Fatalf("follow-up messages = %d, want 4", len(followUpMessages))
	}
	assistant := followUpMessages[2].(map[string]any)
	if assistant["role"] != "assistant" {
		t.Fatalf("assistant role = %v", assistant["role"])
	}
	assistantCalls := assistant["tool_calls"].([]any)
	assistantCall := assistantCalls[0].(map[string]any)
	if assistantCall["type"] != "function" {
		t.Fatalf("assistant tool call type = %v, want function", assistantCall["type"])
	}
	toolResult := followUpMessages[3].(map[string]any)
	if toolResult["role"] != "tool" || toolResult["tool_call_id"] != "call-1" {
		t.Fatalf("tool result message = %#v", toolResult)
	}
}

func TestProvider_GenerateNormalizesBuiltInToolSchemas(t *testing.T) {
	var body map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer ts.Close()

	planProperties := map[string]any{
		"steps": map[string]any{
			"type":        "array",
			"description": "ordered plan steps",
		},
	}
	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithModel("minimaxai/minimax-m3"),
	)
	_, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "plan and delegate",
		Tools: []ai.Tool{
			{Name: "task_TaskService_Add", Description: "add task", Properties: map[string]any{"title": map[string]any{"type": "string"}}},
			{Name: "plan", Description: "record a plan", Properties: planProperties},
			{Name: "request_input", Description: "request input", Properties: map[string]any{"prompt": map[string]any{"type": "string"}}},
			{Name: "delegate", Description: "delegate work", Properties: map[string]any{"task": map[string]any{"type": "string"}, "to": map[string]any{"type": "string"}}},
		},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	tools := body["tools"].([]any)
	if len(tools) != 4 {
		t.Fatalf("tools = %d, want custom tool plus built-ins", len(tools))
	}
	planTool := tools[1].(map[string]any)
	fn := planTool["function"].(map[string]any)
	params := fn["parameters"].(map[string]any)
	props := params["properties"].(map[string]any)
	steps := props["steps"].(map[string]any)
	if _, ok := steps["items"].(map[string]any); !ok {
		t.Fatalf("plan steps schema = %#v, want array items for AtlasCloud/minimax", steps)
	}
	if _, mutated := planProperties["steps"].(map[string]any)["items"]; mutated {
		t.Fatalf("Generate mutated caller tool schema: %#v", planProperties)
	}
}

func TestProvider_GenerateExecutesFollowUpToolCall(t *testing.T) {
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(bodies) {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-1","function":{"name":"conformance_echo","arguments":"{\"value\":\"agent-conformance\"}"}}]}}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-2","function":{"name":"delegate","arguments":"{\"task\":\"summarize the conformance marker\",\"to\":\"blocked-reviewer\"}"}}]}}]}`))
		case 3:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"blocked by policy"}}]}`))
		default:
			t.Fatalf("unexpected API call %d", len(bodies))
		}
	}))
	defer ts.Close()

	var sawEcho, sawDelegate bool
	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithToolHandler(func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
			switch call.Name {
			case "conformance_echo":
				sawEcho = true
				return ai.ToolResult{ID: call.ID, Content: `{"marker":"agent-conformance-ok"}`}
			case "delegate":
				sawDelegate = true
				return ai.ToolResult{ID: call.ID, Refused: ai.RefusedApproval, Content: "blocked by policy"}
			default:
				t.Fatalf("unexpected tool call %+v", call)
				return ai.ToolResult{}
			}
		}),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "run conformance",
		Tools: []ai.Tool{
			{Name: "conformance_echo", Description: "echo conformance marker", Properties: map[string]any{"value": map[string]any{"type": "string"}}},
			{Name: "delegate", Description: "delegate work", Properties: map[string]any{"task": map[string]any{"type": "string"}, "to": map[string]any{"type": "string"}}},
		},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !sawEcho || !sawDelegate {
		t.Fatalf("sawEcho=%v sawDelegate=%v, want both tools executed", sawEcho, sawDelegate)
	}
	if len(resp.ToolCalls) != 2 {
		t.Fatalf("ToolCalls = %+v, want echo and delegate", resp.ToolCalls)
	}
	if resp.ToolCalls[1].Name != "delegate" || resp.ToolCalls[1].Error != ai.RefusedApproval {
		t.Fatalf("follow-up delegate = %+v, want refused delegate", resp.ToolCalls[1])
	}
	if !strings.Contains(resp.Answer, "blocked by policy") {
		t.Fatalf("Answer = %q, want follow-up tool result", resp.Answer)
	}
	if !strings.Contains(resp.Answer, "agent-conformance-ok") {
		t.Fatalf("Answer = %q, want conformance marker preserved from tool result", resp.Answer)
	}
	if _, ok := bodies[1]["tools"].([]any); !ok {
		t.Fatalf("follow-up request did not include tools: %#v", bodies[1])
	}
}

func TestProvider_GenerateExecutesMultiStepFollowUpToolCalls(t *testing.T) {
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(bodies) {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-plan","function":{"name":"plan","arguments":"{\"steps\":[{\"task\":\"create tasks\"},{\"task\":\"notify owner\"}]}"}}]}}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-add","function":{"name":"task_TaskService_Add","arguments":"{\"title\":\"Design\"}"}}]}}]}`))
		case 3:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-delegate","function":{"name":"delegate","arguments":"{\"task\":\"notify owner@acme.com\",\"to\":\"comms\"}"}}]}}]}`))
		case 4:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"done"}}]}`))
		default:
			t.Fatalf("unexpected API call %d", len(bodies))
		}
	}))
	defer ts.Close()

	var calls []string
	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithToolHandler(func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
			calls = append(calls, call.Name)
			return ai.ToolResult{ID: call.ID, Content: `{"ok":true}`}
		}),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "plan, create tasks, and delegate notification",
		Tools: []ai.Tool{
			{Name: "plan", Description: "record a plan", Properties: map[string]any{"steps": map[string]any{"type": "array"}}},
			{Name: "task_TaskService_Add", Description: "add task", Properties: map[string]any{"title": map[string]any{"type": "string"}}},
			{Name: "delegate", Description: "delegate work", Properties: map[string]any{"task": map[string]any{"type": "string"}, "to": map[string]any{"type": "string"}}},
		},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	wantCalls := []string{"plan", "task_TaskService_Add", "delegate"}
	if strings.Join(calls, ",") != strings.Join(wantCalls, ",") {
		t.Fatalf("tool calls = %v, want %v", calls, wantCalls)
	}
	if len(resp.ToolCalls) != 3 {
		t.Fatalf("ToolCalls = %+v, want all multi-step calls", resp.ToolCalls)
	}
	if resp.Answer != "done" {
		t.Fatalf("Answer = %q, want final follow-up reply", resp.Answer)
	}
	if len(bodies) != 4 {
		t.Fatalf("requests = %d, want initial plus three follow-ups", len(bodies))
	}
	for i := 1; i < 4; i++ {
		if _, ok := bodies[i]["tools"].([]any); !ok {
			t.Fatalf("follow-up request %d did not include tools: %#v", i+1, bodies[i])
		}
	}
}

func TestProvider_GeneratePreservesFollowUpTextToolCallInReply(t *testing.T) {
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(bodies) {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-1","function":{"name":"conformance_echo","arguments":"{\"value\":\"agent-conformance\"}"}}]}}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<tool_call name=\"delegate\">{\"task\":\"summarize the conformance marker\",\"to\":\"blocked-reviewer\"}</tool_call>"}}]}`))
		default:
			t.Fatalf("unexpected API call %d", len(bodies))
		}
	}))
	defer ts.Close()

	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithToolHandler(func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
			if call.Name != "conformance_echo" {
				t.Fatalf("unexpected structured tool call %+v", call)
			}
			return ai.ToolResult{ID: call.ID, Content: `{"marker":"agent-conformance-ok"}`}
		}),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "run conformance",
		Tools: []ai.Tool{
			{Name: "conformance_echo", Description: "echo conformance marker", Properties: map[string]any{"value": map[string]any{"type": "string"}}},
			{Name: "delegate", Description: "delegate work", Properties: map[string]any{"task": map[string]any{"type": "string"}, "to": map[string]any{"type": "string"}}},
		},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !strings.Contains(resp.Reply, `<tool_call name="delegate">`) {
		t.Fatalf("Reply = %q, want tagged delegate follow-up for agent text fallback", resp.Reply)
	}
	if resp.Answer != "" {
		t.Fatalf("Answer = %q, want follow-up text preserved only as Reply", resp.Answer)
	}
}

func TestProvider_GenerateRepairsInitialPartialTextToolCall(t *testing.T) {
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(bodies) {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<tool_call name=\"plan\">"}}]}`))
		case 2:
			messages := body["messages"].([]any)
			last := messages[len(messages)-1].(map[string]any)
			if last["role"] != "user" || !strings.Contains(last["content"].(string), "did not finish valid tool-call markup") {
				t.Fatalf("repair prompt = %#v, want partial tool-call guidance", last)
			}
			if _, ok := body["tools"]; !ok {
				t.Fatalf("repair request did not keep tools available: %#v", body)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<tool_call name=\"plan\">{\"steps\":[{\"task\":\"create tasks\"}]}</tool_call>"}}]}`))
		default:
			t.Fatalf("unexpected API call %d", len(bodies))
		}
	}))
	defer ts.Close()

	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithModel("minimaxai/minimax-m3"),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "plan and delegate",
		Tools: []ai.Tool{
			{Name: "plan", Description: "record a plan", Properties: map[string]any{"steps": map[string]any{"type": "array"}}},
			{Name: "delegate", Description: "delegate work", Properties: map[string]any{"task": map[string]any{"type": "string"}}},
		},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !strings.Contains(resp.Reply, `<tool_call name="plan">`) || !strings.Contains(resp.Reply, `</tool_call>`) {
		t.Fatalf("Reply = %q, want completed text tool call", resp.Reply)
	}
	if len(bodies) != 2 {
		t.Fatalf("requests = %d, want initial plus repair", len(bodies))
	}
}

func TestProvider_GenerateFallsBackAfterRepeatedPartialPlanTextToolCall(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<tool_call name=\"plan\">"}}]}`))
	}))
	defer ts.Close()

	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithModel("minimaxai/minimax-m3"),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "plan and delegate",
		Tools:  []ai.Tool{{Name: "plan", Description: "record a plan"}},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !strings.Contains(resp.Reply, `<tool_call name="plan">`) || !strings.Contains(resp.Reply, `</tool_call>`) {
		t.Fatalf("Reply = %q, want completed fallback plan text tool call", resp.Reply)
	}
	if !strings.Contains(resp.Reply, "plan and delegate") {
		t.Fatalf("Reply = %q, want fallback plan seeded from prompt", resp.Reply)
	}
}

func TestProvider_GenerateFallsBackAfterRepeatedPartialDelegateTextToolCall(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<tool_call name=\"delegate\">"}}]}`))
	}))
	defer ts.Close()

	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithModel("minimaxai/minimax-m3"),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		SystemPrompt: "You coordinate launch work and delegate readiness notifications to the comms agent.",
		Prompt:       "delegate the owner readiness notification to comms",
		Tools:        []ai.Tool{{Name: "delegate", Description: "delegate work"}},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	for _, want := range []string{`<tool_call name="delegate">`, `"task":"delegate the owner readiness notification to comms"`, `"to":"comms"`, `</tool_call>`} {
		if !strings.Contains(resp.Reply, want) {
			t.Fatalf("Reply = %q, want delegate fallback containing %q", resp.Reply, want)
		}
	}
}

func TestProvider_GenerateFallsBackAfterRepeatedPartialNoArgumentServiceTextToolCall(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<tool_call name=\"task_TaskService_List\">"}}]}`))
	}))
	defer ts.Close()

	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithModel("minimaxai/minimax-m3"),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "list the current launch-readiness tasks",
		Tools: []ai.Tool{{
			Name:         "task_TaskService_List",
			OriginalName: "task.TaskService.List",
			Description:  "List persisted launch-readiness tasks",
			Properties:   map[string]any{},
		}},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	want := `<tool_call name="task_TaskService_List">{}</tool_call>`
	if resp.Reply != want {
		t.Fatalf("Reply = %q, want %q", resp.Reply, want)
	}
}

func TestProvider_GenerateFallsBackAfterRepeatedPartialWorkspaceServiceTextToolCall(t *testing.T) {
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<tool_call name=\"workspace_WorkspaceService_Create\">"}}]}`))
	}))
	defer ts.Close()

	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithModel("minimaxai/minimax-m3"),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		SystemPrompt: "Create an onboarding workspace only if it is still needed.",
		Prompt:       "Onboard alice@acme.com. The workspace create side effect may already be complete; avoid failing the flow on a duplicate repaired call.",
		Tools: []ai.Tool{{
			Name:         "workspace_WorkspaceService_Create",
			OriginalName: "workspace.WorkspaceService.Create",
			Description:  "Create an onboarding workspace",
			Properties:   map[string]any{"owner": map[string]any{"type": "string"}},
		}},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	want := `<tool_call name="workspace_WorkspaceService_Create">{"owner":"alice@acme.com"}</tool_call>`
	if resp.Reply != want {
		t.Fatalf("Reply = %q, want %q", resp.Reply, want)
	}
	if len(bodies) != 2 {
		t.Fatalf("requests = %d, want initial plus repair", len(bodies))
	}
}

func TestProvider_GenerateRetriesMinimaxBuiltInsAsTextTools(t *testing.T) {
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(bodies) {
		case 1:
			http.Error(w, `{"code":400,"msg":"bad request"}`, http.StatusBadRequest)
		case 2:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<tool_call name=\"delegate\">{\"task\":\"summarize\",\"to\":\"blocked-reviewer\"}</tool_call>"}}]}`))
		default:
			t.Fatalf("unexpected API call %d", len(bodies))
		}
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL), ai.WithModel("minimaxai/minimax-m3"))
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "plan and delegate",
		Tools: []ai.Tool{
			{Name: "task_TaskService_Add", Description: "add task", Properties: map[string]any{"title": map[string]any{"type": "string"}}},
			{Name: "plan", Description: "record a plan", Properties: map[string]any{"steps": map[string]any{"type": "array"}}},
			{Name: "request_input", Description: "request input", Properties: map[string]any{"prompt": map[string]any{"type": "string"}}},
			{Name: "delegate", Description: "delegate work", Properties: map[string]any{"task": map[string]any{"type": "string"}, "to": map[string]any{"type": "string"}}},
		},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !strings.Contains(resp.Reply, `<tool_call name="delegate">`) {
		t.Fatalf("Reply = %q, want text delegate fallback", resp.Reply)
	}
	if len(bodies) != 2 {
		t.Fatalf("requests = %d, want initial plus compat retry", len(bodies))
	}
	initialTools := bodies[0]["tools"].([]any)
	if len(initialTools) != 4 {
		t.Fatalf("initial tools = %d, want all tools", len(initialTools))
	}
	retryTools := bodies[1]["tools"].([]any)
	if len(retryTools) != 1 {
		t.Fatalf("retry tools = %d, want only service tools", len(retryTools))
	}
	fn := retryTools[0].(map[string]any)["function"].(map[string]any)
	if fn["name"] != "task_TaskService_Add" {
		t.Fatalf("retry tool name = %v, want service tool only", fn["name"])
	}
	msgs := bodies[1]["messages"].([]any)
	compat := msgs[len(msgs)-1].(map[string]any)
	if compat["role"] != "system" || !strings.Contains(compat["content"].(string), `<tool_call name="tool_name">`) {
		t.Fatalf("compat instruction = %#v", compat)
	}
}

func TestProvider_GenerateRetriesMinimaxServiceToolsAsTextTools(t *testing.T) {
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(bodies) {
		case 1:
			http.Error(w, `{"code":400,"msg":"bad request"}`, http.StatusBadRequest)
		case 2:
			if _, ok := body["tools"]; ok {
				t.Fatalf("text-tool retry included native tools: %#v", body["tools"])
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<tool_call name=\"conformance_echo\">{\"value\":\"agent-conformance\"}</tool_call>"}}]}`))
		default:
			t.Fatalf("unexpected API call %d", len(bodies))
		}
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL), ai.WithModel("minimaxai/minimax-m3"))
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "call a tool",
		Tools: []ai.Tool{{
			Name:        "conformance_echo",
			Description: "echo conformance marker",
			Properties:  map[string]any{"value": map[string]any{"type": "string"}},
		}},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !strings.Contains(resp.Reply, `<tool_call name="conformance_echo">`) {
		t.Fatalf("Reply = %q, want text service-tool fallback", resp.Reply)
	}
	if len(bodies) != 2 {
		t.Fatalf("requests = %d, want initial plus text-tool retry", len(bodies))
	}
	if _, ok := bodies[0]["tools"].([]any); !ok {
		t.Fatalf("initial request did not include native tools: %#v", bodies[0])
	}
	msgs := bodies[1]["messages"].([]any)
	compat := msgs[len(msgs)-1].(map[string]any)
	content := compat["content"].(string)
	for _, want := range []string{"native tools payload was rejected", `<tool_call name="tool_name">`, "conformance_echo"} {
		if !strings.Contains(content, want) {
			t.Fatalf("text-tool instruction %q missing %q", content, want)
		}
	}
}

func TestProvider_GenerateFollowUpRetriesWithoutToolsOnBadRequest(t *testing.T) {
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(bodies) {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-1","function":{"name":"conformance_echo","arguments":"{\"value\":\"agent-conformance\"}"}}]}}]}`))
		case 2:
			http.Error(w, `{"code":400,"msg":"bad request"}`, http.StatusBadRequest)
		case 3:
			if _, ok := body["tools"]; ok {
				t.Fatalf("no-tools retry still included tools: %#v", body["tools"])
			}
			messages := body["messages"].([]any)
			last := messages[len(messages)-1].(map[string]any)
			if last["role"] == "tool" {
				http.Error(w, `{"code":400,"msg":"trailing tool message rejected"}`, http.StatusBadRequest)
				return
			}
			if last["role"] != "user" || !strings.Contains(last["content"].(string), "Tool result for call-1") {
				t.Fatalf("no-tools retry last message = %#v, want user-visible tool result", last)
			}
			assistant := messages[len(messages)-2].(map[string]any)
			if _, ok := assistant["tool_calls"]; ok {
				t.Fatalf("no-tools retry assistant still included tool_calls: %#v", assistant)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"done"}}]}`))
		default:
			t.Fatalf("unexpected API call %d", len(bodies))
		}
	}))
	defer ts.Close()

	var toolCalls int
	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithModel("minimaxai/minimax-m3"),
		ai.WithToolHandler(func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
			toolCalls++
			return ai.ToolResult{ID: call.ID, Content: `{"marker":"agent-conformance-ok"}`}
		}),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "call a tool",
		Tools:  []ai.Tool{{Name: "conformance_echo", Description: "echo", Properties: map[string]any{"value": map[string]any{"type": "string"}}}},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.Answer != "done" {
		t.Fatalf("Answer = %q, want done", resp.Answer)
	}
	if toolCalls != 1 {
		t.Fatalf("tool handler calls = %d, want one (no duplicate side effect)", toolCalls)
	}
	if len(bodies) != 3 {
		t.Fatalf("requests = %d, want chat, failed follow-up, no-tools follow-up", len(bodies))
	}
	if _, ok := bodies[1]["tools"]; !ok {
		t.Fatalf("first follow-up did not include tools")
	}
}

func TestProvider_GenerateToolCallHTTPErrorIncludesRequestContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":400,"msg":"bad request"}`, http.StatusBadRequest)
	}))
	defer ts.Close()

	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithModel("deepseek-ai/DeepSeek-V3-0324"),
	)
	_, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "call a tool",
		Tools: []ai.Tool{{
			Name:        "conformance_echo",
			Description: "echo conformance marker",
			Properties:  map[string]any{"value": map[string]any{"type": "string"}},
		}},
	})
	if err == nil {
		t.Fatal("Generate error = nil, want 400")
	}
	msg := err.Error()
	for _, want := range []string{"400 Bad Request", "atlascloud chat request", "model=deepseek-ai/DeepSeek-V3-0324", "tools=1", "tool_names=conformance_echo"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
	if strings.Contains(msg, "test-key") {
		t.Fatalf("error leaked API key: %s", msg)
	}
}

func TestProvider_Registration(t *testing.T) {
	m := ai.New("atlascloud", ai.WithAPIKey("test"))
	if m == nil {
		t.Fatal("ai.New('atlascloud') returned nil — provider not registered")
	}
	if m.String() != "atlascloud" {
		t.Errorf("Expected 'atlascloud', got '%s'", m.String())
	}
}

func TestProvider_ImageRegistration(t *testing.T) {
	ig := ai.NewImage("atlascloud", ai.WithAPIKey("test"))
	if ig == nil {
		t.Fatal("ai.NewImage('atlascloud') returned nil — image provider not registered")
	}
	if ig.String() != "atlascloud" {
		t.Errorf("Expected 'atlascloud', got '%s'", ig.String())
	}
}

func TestProvider_GenerateImage_NoAPIKey(t *testing.T) {
	p := NewProvider()
	_, err := p.GenerateImage(context.Background(), &ai.ImageRequest{Prompt: "a cat"})
	if err == nil {
		t.Error("Expected error when API key is missing, got nil")
	}
}

func TestProvider_ImplementsImageModel(t *testing.T) {
	var _ ai.ImageModel = (*Provider)(nil)
}

func TestProvider_VideoRegistration(t *testing.T) {
	vg := ai.NewVideo("atlascloud", ai.WithAPIKey("test"))
	if vg == nil {
		t.Fatal("ai.NewVideo('atlascloud') returned nil — video provider not registered")
	}
	if vg.String() != "atlascloud" {
		t.Errorf("Expected 'atlascloud', got '%s'", vg.String())
	}
}

func TestProvider_GenerateVideo_NoAPIKey(t *testing.T) {
	p := NewProvider()
	_, err := p.GenerateVideo(context.Background(), &ai.VideoRequest{Prompt: "a cat"})
	if err == nil {
		t.Error("Expected error when API key is missing, got nil")
	}
}

func TestProvider_ImplementsVideoModel(t *testing.T) {
	var _ ai.VideoModel = (*Provider)(nil)
}
