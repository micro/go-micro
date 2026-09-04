package openaiapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(code int, body string) roundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: code,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}
}

type requestRecord struct {
	method string
	path   string
	header http.Header
	body   []byte
}

type recorder struct {
	calls []requestRecord
	next  http.RoundTripper
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(strings.NewReader(string(body)))
	r.calls = append(r.calls, requestRecord{
		method: req.Method, path: req.URL.Path, header: req.Header.Clone(), body: body,
	})
	if r.next == nil {
		return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("no next transport"))}, nil
	}
	return r.next.RoundTrip(req)
}

func (r *recorder) decode(i int) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(r.calls[i].body, &m); err != nil {
		panic(err)
	}
	return m
}

type seqRoundTripper struct {
	responses []roundTripFunc
	idx       int
}

func (s *seqRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if s.idx >= len(s.responses) {
		return jsonResponse(500, "unexpected extra request")(r)
	}
	rr := s.responses[s.idx]
	s.idx++
	return rr(r)
}

func demoClient(t *testing.T, cfg Config, rt http.RoundTripper, opts ...ai.Option) *Client {
	t.Helper()
	return New(cfg, append(opts, ai.WithTransport(rt))...)
}

func TestNew_FoldsDefaults(t *testing.T) {
	c := New(Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "demo-model"})
	if c.String() != "demo" {
		t.Fatalf("String = %q", c.String())
	}
	o := c.Options()
	if o.Model != "demo-model" {
		t.Fatalf("model = %q, want default", o.Model)
	}
	if o.BaseURL != "https://x.example" {
		t.Fatalf("baseURL = %q, want default", o.BaseURL)
	}
	if err := c.Init(ai.WithModel("other")); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if c.Options().Model != "other" {
		t.Fatalf("model after Init = %q", c.Options().Model)
	}
}

func TestGenerate_SingleTurn(t *testing.T) {
	rec := &recorder{next: jsonResponse(200, `{"choices":[{"message":{"content":"hi"}}]}`)}
	c := demoClient(t, Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m"},
		rec, ai.WithAPIKey("k"))

	resp, err := c.Generate(context.Background(), &ai.Request{
		SystemPrompt: "sys",
		Prompt:       "hello",
		Messages:     []ai.Message{{Role: "user", Content: "previous"}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Reply != "hi" {
		t.Fatalf("Reply = %q", resp.Reply)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(rec.calls))
	}
	call := rec.calls[0]
	if call.method != http.MethodPost || call.path != "/v1/chat/completions" {
		t.Fatalf("call = %s %s", call.method, call.path)
	}
	if got := call.header.Get("Authorization"); got != "Bearer k" {
		t.Fatalf("Authorization = %q", got)
	}
	body := rec.decode(0)
	if body["model"] != "m" {
		t.Fatalf("model = %v", body["model"])
	}
	messages := body["messages"].([]any)
	if len(messages) != 2 {
		t.Fatalf("messages = %#v, want system+user only (single-turn)", messages)
	}
	if messages[0].(map[string]any)["role"] != "system" || messages[1].(map[string]any)["role"] != "user" {
		t.Fatalf("messages roles wrong: %#v", messages)
	}
}

func TestGenerate_UseMessages(t *testing.T) {
	for _, tc := range []struct {
		name   string
		useMsg bool
	}{
		{name: "off", useMsg: false},
		{name: "on", useMsg: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recorder{next: jsonResponse(200, `{"choices":[{"message":{"content":"hi"}}]}`)}
			c := demoClient(t, Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m", UseMessages: tc.useMsg},
				rec, ai.WithAPIKey("k"), ai.WithMaxTokens(8), ai.WithEffort("high"))

			if _, err := c.Generate(context.Background(), &ai.Request{
				SystemPrompt: "sys",
				Prompt:       "hello",
				Messages:     []ai.Message{{Role: "user", Content: "previous"}},
			}); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			body := rec.decode(0)

			if tc.useMsg {
				messages := body["messages"].([]any)
				if len(messages) != 3 {
					t.Fatalf("messages = %#v, want system+history+prompt", messages)
				}
				if r := messages[1].(map[string]any)["role"]; r != "user" {
					t.Fatalf("message[1] role = %v", r)
				}
				if got := messages[1].(map[string]any)["content"]; got != "previous" {
					t.Fatalf("message[1] content = %v, want history preserved", got)
				}
				if body["max_tokens"] != float64(8) {
					t.Fatalf("max_tokens = %v", body["max_tokens"])
				}
				if body["reasoning_effort"] != "high" {
					t.Fatalf("reasoning_effort = %v", body["reasoning_effort"])
				}
			} else {
				if body["max_tokens"] != nil || body["reasoning_effort"] != nil {
					t.Fatalf("single-turn must not send max_tokens/effort: %#v", body)
				}
				messages := body["messages"].([]any)
				if len(messages) != 2 {
					t.Fatalf("messages = %#v, want system+user only", messages)
				}
			}
		})
	}
}

func TestGenerate_ToolLoop(t *testing.T) {
	seq := &seqRoundTripper{responses: []roundTripFunc{
		jsonResponse(200, `{"choices":[{"message":{"content":"","tool_calls":[
			{"id":"call_1","function":{"name":"lookup","arguments":"{\"q\":\"a\"}"}}]}}]}`),
		jsonResponse(200, `{"choices":[{"message":{"content":"final answer"}}]}`),
	}}
	rec := &recorder{next: seq}
	c := demoClient(t, Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m"},
		rec, ai.WithAPIKey("k"))

	var called string
	handler := func(ctx context.Context, tc ai.ToolCall) ai.ToolResult {
		called = tc.Name
		if tc.ID != "call_1" {
			t.Fatalf("tool call ID = %q", tc.ID)
		}
		if v, ok := tc.Input["q"].(string); !ok || v != "a" {
			t.Fatalf("tool input = %#v", tc.Input)
		}
		return ai.ToolResult{ID: tc.ID, Content: `{"ok":1}`}
	}
	if err := c.Init(func(o *ai.Options) { o.ToolHandler = handler }); err != nil {
		t.Fatalf("Init: %v", err)
	}

	resp, err := c.Generate(context.Background(), &ai.Request{
		SystemPrompt: "sys",
		Prompt:       "hello",
		Tools:        []ai.Tool{{Name: "lookup", Description: "looks up", Properties: map[string]any{"q": map[string]any{"type": "string"}}}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if called != "lookup" {
		t.Fatalf("handler called with %q, want lookup", called)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Result != `{"ok":1}` {
		t.Fatalf("ToolCalls = %#v, want Result populated", resp.ToolCalls)
	}
	if resp.Answer != "final answer" {
		t.Fatalf("Answer = %q", resp.Answer)
	}
	if len(rec.calls) != 2 {
		t.Fatalf("calls = %d, want first + follow-up", len(rec.calls))
	}
	first := rec.decode(0)
	tools := first["tools"].([]any)
	tf := tools[0].(map[string]any)
	if tf["type"] != "function" {
		t.Fatalf("tools[0] = %#v", tf)
	}
	if _, ok := tf["function"].(map[string]any); !ok {
		t.Fatalf("tools[0].function missing: %#v", tf)
	}
	follow := rec.decode(1)
	msgs := follow["messages"].([]any)
	last := msgs[len(msgs)-1].(map[string]any)
	if last["role"] != "tool" || last["tool_call_id"] != "call_1" {
		t.Fatalf("follow-up last message = %#v, want tool result", last)
	}
}

func TestGenerate_ParseUsage(t *testing.T) {
	body := `{"choices":[{"message":{"content":"hi"}}],"usage":
		{"prompt_tokens":7,"completion_tokens":5,"total_tokens":12}}`
	for _, tc := range []struct {
		name      string
		parse     bool
		wantInput int
	}{
		{name: "parsed", parse: true, wantInput: 7},
		{name: "unparsed", parse: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := demoClient(t, Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m", ParseUsage: tc.parse},
				jsonResponse(200, body), ai.WithAPIKey("k"))
			resp, err := c.Generate(context.Background(), &ai.Request{Prompt: "hi"})
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if resp.Usage.InputTokens != tc.wantInput {
				t.Fatalf("Usage = %+v, want InputTokens=%d", resp.Usage, tc.wantInput)
			}
		})
	}
}

func TestGenerate_HTTPError(t *testing.T) {
	c := demoClient(t, Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m"},
		roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": {"1"}},
				Body: io.NopCloser(strings.NewReader("rate limited"))}, nil
		}), ai.WithAPIKey("k"))

	_, err := c.Generate(context.Background(), &ai.Request{Prompt: "hi"})
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	var he *ai.HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("error = %T, want *ai.HTTPError", err)
	}
	if he.StatusCode() != 429 {
		t.Fatalf("status = %d", he.StatusCode())
	}
	if d := he.RetryAfter(); d != time.Second {
		t.Fatalf("RetryAfter = %v", d)
	}
	if ai.ClassifyError(err) != ai.ErrorKindRateLimited {
		t.Fatalf("classify = %q, want rate_limited", ai.ClassifyError(err))
	}
}

func TestStream_SSE(t *testing.T) {
	sse := "data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n" +
		"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5}}\n\n" +
		"data: [DONE]\n\n"
	rec := &recorder{next: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(sse))}, nil
	})}
	c := demoClient(t, Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m"},
		rec, ai.WithAPIKey("k"))

	s, err := c.Stream(context.Background(), &ai.Request{
		SystemPrompt: "sys",
		Messages:     []ai.Message{{Role: "user", Content: "prev"}, {Role: "assistant", Content: "ans"}},
		Prompt:       "p",
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer s.Close()

	assertReply(t, s, "hel")
	assertReply(t, s, "lo")
	usage, err := s.Recv()
	if err != nil {
		t.Fatalf("usage chunk: %v", err)
	}
	if usage.Usage != (ai.Usage{InputTokens: 3, OutputTokens: 2, TotalTokens: 5}) {
		t.Fatalf("usage = %+v", usage.Usage)
	}
	if _, err := s.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("final = %v, want EOF", err)
	}

	body := rec.decode(0)
	if body["stream"] != true {
		t.Fatalf("stream = %v", body["stream"])
	}
	so, ok := body["stream_options"].(map[string]any)
	if !ok || so["include_usage"] != true {
		t.Fatalf("stream_options = %v", body["stream_options"])
	}
	if got := rec.calls[0].header.Get("Accept"); got != "text/event-stream" {
		t.Fatalf("Accept = %q", got)
	}
	if msgs := body["messages"].([]any); len(msgs) != 4 {
		t.Fatalf("messages = %#v, want system+history+prompt", msgs)
	}
}

func TestStream_HTTPError(t *testing.T) {
	c := demoClient(t, Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m"},
		roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": {"2"}},
				Body: io.NopCloser(strings.NewReader("quota"))}, nil
		}), ai.WithAPIKey("k"))

	_, err := c.Stream(context.Background(), &ai.Request{Prompt: "hi"})
	if err == nil {
		t.Fatal("Stream returned nil error")
	}
	var he *ai.HTTPError
	if !errors.As(err, &he) || he.StatusCode() != 429 {
		t.Fatalf("error = %T %v, want *ai.HTTPError 429", err, err)
	}
	if ai.ClassifyError(err) != ai.ErrorKindRateLimited {
		t.Fatalf("classify = %q", ai.ClassifyError(err))
	}
}

func TestStream_MalformedChunk(t *testing.T) {
	c := demoClient(t, Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m"},
		roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}},
				Body: io.NopCloser(strings.NewReader("data: {bad json}\n\n"))}, nil
		}), ai.WithAPIKey("k"))

	s, err := c.Stream(context.Background(), &ai.Request{Prompt: "hi"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer s.Close()
	if _, err := s.Recv(); err == nil {
		t.Fatal("Recv returned nil error for malformed chunk")
	}
}

// TestProviderLiteralsConform locks the five real provider Config literals so
// a drift in any shell's defaults or chat path fails here.
func TestProviderLiteralsConform(t *testing.T) {
	cases := []struct {
		name  string
		cfg   Config
		model string
		base  string
		multi bool
		usage bool
	}{
		{name: "openai", model: "gpt-4o", base: "https://api.openai.com", multi: true, usage: true,
			cfg: Config{Name: "openai", DefaultBase: "https://api.openai.com", DefaultModel: "gpt-4o", UseMessages: true, ParseUsage: true}},
		{name: "groq", model: "llama-3.3-70b-versatile", base: "https://api.groq.com/openai",
			cfg: Config{Name: "groq", DefaultBase: "https://api.groq.com/openai", DefaultModel: "llama-3.3-70b-versatile"}},
		{name: "mistral", model: "mistral-large-latest", base: "https://api.mistral.ai",
			cfg: Config{Name: "mistral", DefaultBase: "https://api.mistral.ai", DefaultModel: "mistral-large-latest"}},
		{name: "together", model: "meta-llama/Llama-3.3-70B-Instruct-Turbo", base: "https://api.together.xyz",
			cfg: Config{Name: "together", DefaultBase: "https://api.together.xyz", DefaultModel: "meta-llama/Llama-3.3-70B-Instruct-Turbo"}},
		{name: "minimax", model: "MiniMax-M3", base: "https://api.minimax.io", multi: true,
			cfg: Config{Name: "minimax", DefaultBase: "https://api.minimax.io", DefaultModel: "MiniMax-M3", UseMessages: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recorder{next: jsonResponse(http.StatusOK, `{"choices":[{"message":{"content":"hi"}}]}`)}
			c := New(tc.cfg, ai.WithAPIKey("k"), ai.WithTransport(rec))
			if c.String() != tc.name {
				t.Fatalf("String = %q", c.String())
			}
			o := c.Options()
			if o.Model != tc.model || o.BaseURL != tc.base {
				t.Fatalf("defaults = %q/%q, want %q/%q", o.Model, o.BaseURL, tc.model, tc.base)
			}
			resp, err := c.Generate(context.Background(), &ai.Request{SystemPrompt: "sys", Prompt: "hello", Messages: []ai.Message{{Role: "user", Content: "prev"}}})
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if resp.Reply != "hi" {
				t.Fatalf("Reply = %q", resp.Reply)
			}
			body := rec.decode(0)
			if body["model"] != tc.model {
				t.Fatalf("request model = %v", body["model"])
			}
			if got := rec.calls[0].header.Get("Authorization"); got != "Bearer k" {
				t.Fatalf("Authorization = %q", got)
			}
			if msgs := len(body["messages"].([]any)); (tc.multi && msgs != 3) || (!tc.multi && msgs != 2) {
				t.Fatalf("messages = %d, want %d", msgs, map[bool]int{true: 3, false: 2}[tc.multi])
			}
		})
	}
}

func TestGenerate_HTTPError_Retryable(t *testing.T) {
	c := demoClient(t, Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m"},
		roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": {"1"}},
				Body: io.NopCloser(strings.NewReader("rate limited"))}, nil
		}), ai.WithAPIKey("k"))

	_, err := ai.GenerateWithRetry(context.Background(), c, &ai.Request{Prompt: "hi"},
		ai.GeneratePolicy{MaxAttempts: 2, Backoff: time.Millisecond})
	if err == nil {
		t.Fatal("GenerateWithRetry returned nil error")
	}
	var re *ai.RetryError
	if !errors.As(err, &re) {
		t.Fatalf("error = %T, want *ai.RetryError", err)
	}
	if re.ErrorKind() != ai.ErrorKindRateLimited {
		t.Fatalf("kind = %q, want rate_limited", re.ErrorKind())
	}
}

func TestGenerateImage(t *testing.T) {
	c := demoClient(t, Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m"},
		jsonResponse(200, `{"data":[{"url":"u","b64_json":"b"}]}`), ai.WithAPIKey("k"))
	resp, err := c.GenerateImage(context.Background(), &ai.ImageRequest{Prompt: "a cat"})
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if len(resp.Images) != 1 || resp.Images[0].URL != "u" || resp.Images[0].Base64 != "b" {
		t.Fatalf("Images = %#v", resp.Images)
	}
}

func TestGenerateImage_HTTPError(t *testing.T) {
	c := demoClient(t, Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m"},
		jsonResponse(402, `{"error":"payment required"}`), ai.WithAPIKey("k"))
	_, err := c.GenerateImage(context.Background(), &ai.ImageRequest{Prompt: "a cat"})
	var he *ai.HTTPError
	if !errors.As(err, &he) || he.StatusCode() != 402 {
		t.Fatalf("error = %T %v, want *ai.HTTPError 402", err, err)
	}
}

func TestEmptyRequestNoAPIKeyFails(t *testing.T) {
	c := New(Config{Name: "demo", DefaultBase: "https://x.example", DefaultModel: "m"},
		ai.WithTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("Authorization") == "Bearer " {
				return &http.Response{StatusCode: http.StatusUnauthorized,
					Body: io.NopCloser(strings.NewReader("missing key"))}, nil
			}
			return jsonResponse(http.StatusOK, `{"choices":[{"message":{"content":"hi"}}]}`)(r)
		})))

	_, err := c.Generate(context.Background(), &ai.Request{Prompt: "hi"})
	if err == nil {
		t.Fatal("expected error without API key")
	}
	var he *ai.HTTPError
	if !errors.As(err, &he) || he.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("error = %T %v, want unauthorized", err, err)
	}
}

func assertReply(t *testing.T, s ai.Stream, want string) {
	t.Helper()
	chunk, err := s.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if chunk.Reply != want {
		t.Fatalf("Reply = %q, want %q", chunk.Reply, want)
	}
}
