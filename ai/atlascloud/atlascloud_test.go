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
	if _, ok := bodies[1]["tools"].([]any); !ok {
		t.Fatalf("follow-up request did not include tools: %#v", bodies[1])
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

func TestProvider_GenerateToolCallHTTPErrorIncludesRequestContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":400,"msg":"bad request"}`, http.StatusBadRequest)
	}))
	defer ts.Close()

	p := NewProvider(
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL(ts.URL),
		ai.WithModel("minimaxai/minimax-m3"),
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
	for _, want := range []string{"400 Bad Request", "atlascloud chat request", "model=minimaxai/minimax-m3", "tools=1", "tool_names=conformance_echo"} {
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
