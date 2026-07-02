// Package ollama implements the Ollama model provider.
//
// Ollama runs open-weight models locally (or via Ollama Cloud). This
// provider supports two API styles:
//
//   - Native (/api/chat):   local Ollama servers (default, http://localhost:11434)
//   - OpenAI-compatible (/v1/chat/completions): Ollama Cloud (https://ollama.com/v1)
//
// The provider auto-detects which style to use based on the base URL.
// Set OLLAMA_BASE_URL to point at your server (local or cloud).
//
// Usage (local):
//
//	import _ "go-micro.dev/v6/ai/ollama"
//
//	m := ai.New("ollama",
//	    ai.WithBaseURL("http://localhost:11434"),
//	    ai.WithModel("llama3.2"),
//	)
//
// Usage (Ollama Cloud):
//
//	m := ai.New("ollama",
//	    ai.WithBaseURL("https://ollama.com/v1"),
//	    ai.WithAPIKey("your-key"),
//	    ai.WithModel("gpt-oss:120b"),
//	)
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-micro.dev/v6/ai"
)

func init() {
	ai.Register("ollama", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
	ai.RegisterStream("ollama")
}

// Provider implements the ai.Model interface for Ollama.
type Provider struct {
	opts ai.Options

	// cloudOverride forces cloud mode for testing. When true, the provider
	// uses the OpenAI-compatible endpoint regardless of the base URL.
	cloudOverride bool
}

// NewProvider creates a new Ollama provider.
func NewProvider(opts ...ai.Option) *Provider {
	options := ai.NewOptions(opts...)
	if options.Model == "" {
		options.Model = "llama3.2"
	}
	if options.BaseURL == "" {
		options.BaseURL = "http://localhost:11434"
	}
	return &Provider{opts: options}
}

// Init initializes the provider with options.
func (p *Provider) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&p.opts)
	}
	return nil
}

// Options returns the provider options.
func (p *Provider) Options() ai.Options { return p.opts }

// String returns the provider name.
func (p *Provider) String() string { return "ollama" }

// isCloud returns true when the base URL points at Ollama Cloud (ollama.com),
// which uses the OpenAI-compatible /v1/chat/completions endpoint instead of
// the native /api/chat.
func (p *Provider) isCloud() bool {
	if p.cloudOverride {
		return true
	}
	return strings.Contains(p.opts.BaseURL, "ollama.com")
}

// chatPath returns the API endpoint path for chat completions.
func (p *Provider) chatPath() string {
	if p.isCloud() {
		return "/v1/chat/completions"
	}
	return "/api/chat"
}

// streamPath returns the API endpoint path for streaming chat.
// Ollama Cloud uses the same /v1/chat/completions with stream:true.
// Local Ollama uses /api/chat with stream:true.
func (p *Provider) streamPath() string {
	return p.chatPath()
}

// Generate generates a response from the Ollama model.
func (p *Provider) Generate(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (*ai.Response, error) {
	if p.isCloud() {
		return p.generateOpenAI(ctx, req)
	}
	return p.generateNative(ctx, req)
}

// Stream generates a streaming response.
func (p *Provider) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
	if p.isCloud() {
		return p.streamOpenAI(ctx, req)
	}
	return p.streamNative(ctx, req)
}

// ---------------------------------------------------------------------------
// OpenAI-compatible mode (Ollama Cloud: ollama.com/v1)
// ---------------------------------------------------------------------------

func (p *Provider) generateOpenAI(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	var tools []map[string]any
	for _, t := range req.Tools {
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters": map[string]any{
					"type":       "object",
					"properties": t.Properties,
				},
			},
		})
	}

	messages := buildOpenAIMessages(req)
	apiReq := map[string]any{
		"model":    p.opts.Model,
		"messages": messages,
		"stream":   false,
	}
	if len(tools) > 0 {
		apiReq["tools"] = tools
	}
	if p.opts.MaxTokens > 0 {
		apiReq["max_tokens"] = p.opts.MaxTokens
	}

	resp, rawMsg, err := p.callOpenAI(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	// No tool calls or no handler — return as-is.
	if len(resp.ToolCalls) == 0 || p.opts.ToolHandler == nil {
		return resp, nil
	}

	// Tool execution loop.
	convMessages := append(messages, map[string]any{
		"role":       "assistant",
		"content":    rawMsg.content,
		"tool_calls": rawMsg.toolCalls,
	})

	pendingCalls := resp.ToolCalls
	for round := 0; round < 10; round++ {
		for i := range pendingCalls {
			result := p.opts.ToolHandler(ctx, pendingCalls[i])
			pendingCalls[i].Result = result.Content
			convMessages = append(convMessages, map[string]any{
				"role":         "tool",
				"tool_call_id": pendingCalls[i].ID,
				"content":      result.Content,
			})
		}

		followUpReq := map[string]any{
			"model":    p.opts.Model,
			"messages": convMessages,
			"stream":   false,
		}
		if len(tools) > 0 {
			followUpReq["tools"] = tools
		}
		if p.opts.MaxTokens > 0 {
			followUpReq["max_tokens"] = p.opts.MaxTokens
		}

		followUpResp, followUpRaw, err := p.callOpenAI(ctx, followUpReq)
		if err != nil {
			break
		}

		if len(followUpResp.ToolCalls) > 0 {
			resp.ToolCalls = append(resp.ToolCalls, followUpResp.ToolCalls...)
			pendingCalls = followUpResp.ToolCalls
			convMessages = append(convMessages, map[string]any{
				"role":       "assistant",
				"content":    followUpRaw.content,
				"tool_calls": followUpRaw.toolCalls,
			})
			continue
		}

		if followUpResp.Reply != "" {
			resp.Answer = followUpResp.Reply
		}
		break
	}

	return resp, nil
}

func (p *Provider) callOpenAI(ctx context.Context, req map[string]any) (*ai.Response, *rawChatMessage, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + p.chatPath()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.opts.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)
	}

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API error (%s): %s", httpResp.Status, string(respBody))
	}

	var chatResp struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Choices []struct {
			Message struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return nil, nil, fmt.Errorf("no response from API")
	}

	choice := chatResp.Choices[0]
	response := &ai.Response{
		Reply: choice.Message.Content,
		Usage: ai.Usage{
			InputTokens:  chatResp.Usage.PromptTokens,
			OutputTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:  chatResp.Usage.TotalTokens,
		},
	}

	var rawToolCalls []map[string]any
	for _, tc := range choice.Message.ToolCalls {
		var input map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
			input = map[string]any{}
		}
		response.ToolCalls = append(response.ToolCalls, ai.ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: input,
		})
		rawToolCalls = append(rawToolCalls, map[string]any{
			"id":   tc.ID,
			"type": "function",
			"function": map[string]any{
				"name":      tc.Function.Name,
				"arguments": tc.Function.Arguments,
			},
		})
	}

	raw := &rawChatMessage{
		content:   choice.Message.Content,
		toolCalls: rawToolCalls,
	}
	return response, raw, nil
}

func (p *Provider) streamOpenAI(ctx context.Context, req *ai.Request) (ai.Stream, error) {
	messages := buildOpenAIMessages(req)
	apiReq := map[string]any{
		"model":          p.opts.Model,
		"messages":       messages,
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	if p.opts.MaxTokens > 0 {
		apiReq["max_tokens"] = p.opts.MaxTokens
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stream request: %w", err)
	}

	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + p.streamPath()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if p.opts.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)
	}

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream API request failed: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("stream API error (%s): %s", httpResp.Status, string(respBody))
	}

	return &sseStream{body: httpResp.Body, scanner: bufio.NewScanner(httpResp.Body)}, nil
}

// buildOpenAIMessages converts an ai.Request into the OpenAI chat message format.
func buildOpenAIMessages(req *ai.Request) []map[string]any {
	messages := []map[string]any{}
	if req.SystemPrompt != "" {
		messages = append(messages, map[string]any{"role": "system", "content": req.SystemPrompt})
	}
	for _, m := range req.Messages {
		messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
	}
	if req.Prompt != "" {
		messages = append(messages, map[string]any{"role": "user", "content": req.Prompt})
	}
	return messages
}

// sseStream reads OpenAI-style server-sent events (used by Ollama Cloud).
type sseStream struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	closed  bool
}

func (s *sseStream) Recv() (*ai.Response, error) {
	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil, io.EOF
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("failed to parse stream chunk: %w", err)
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			return &ai.Response{Reply: chunk.Choices[0].Delta.Content}, nil
		}
		if chunk.Usage != nil {
			return &ai.Response{Usage: ai.Usage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
				TotalTokens:  chunk.Usage.TotalTokens,
			}}, nil
		}
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (s *sseStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.body.Close()
}

// ---------------------------------------------------------------------------
// Native mode (local Ollama: localhost:11434/api/chat)
// ---------------------------------------------------------------------------

func (p *Provider) generateNative(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	var tools []map[string]any
	for _, t := range req.Tools {
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters": map[string]any{
					"type":       "object",
					"properties": t.Properties,
				},
			},
		})
	}

	messages := []map[string]any{}
	if req.SystemPrompt != "" {
		messages = append(messages, map[string]any{"role": "system", "content": req.SystemPrompt})
	}
	for _, m := range req.Messages {
		messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
	}
	if req.Prompt != "" {
		messages = append(messages, map[string]any{"role": "user", "content": req.Prompt})
	}

	apiReq := map[string]any{
		"model":    p.opts.Model,
		"messages": messages,
		"stream":   false,
	}
	if len(tools) > 0 {
		apiReq["tools"] = tools
	}
	if p.opts.MaxTokens > 0 {
		apiReq["options"] = map[string]any{"num_predict": p.opts.MaxTokens}
	}

	resp, rawMsg, err := p.callNative(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	if len(resp.ToolCalls) == 0 || p.opts.ToolHandler == nil {
		return resp, nil
	}

	convMessages := append(messages, map[string]any{
		"role":    "assistant",
		"content": rawMsg.content,
	})
	if len(rawMsg.toolCalls) > 0 {
		convMessages[len(convMessages)-1]["tool_calls"] = rawMsg.toolCalls
	}

	pendingCalls := resp.ToolCalls
	for round := 0; round < 10; round++ {
		for i := range pendingCalls {
			result := p.opts.ToolHandler(ctx, pendingCalls[i])
			pendingCalls[i].Result = result.Content
			convMessages = append(convMessages, map[string]any{
				"role":    "tool",
				"content": result.Content,
			})
		}

		followUpReq := map[string]any{
			"model":    p.opts.Model,
			"messages": convMessages,
			"stream":   false,
		}
		if len(tools) > 0 {
			followUpReq["tools"] = tools
		}
		if p.opts.MaxTokens > 0 {
			followUpReq["options"] = map[string]any{"num_predict": p.opts.MaxTokens}
		}

		followUpResp, followUpRaw, err := p.callNative(ctx, followUpReq)
		if err != nil {
			break
		}

		if len(followUpResp.ToolCalls) > 0 {
			resp.ToolCalls = append(resp.ToolCalls, followUpResp.ToolCalls...)
			pendingCalls = followUpResp.ToolCalls
			convMessages = append(convMessages, map[string]any{
				"role":    "assistant",
				"content": followUpRaw.content,
			})
			if len(followUpRaw.toolCalls) > 0 {
				convMessages[len(convMessages)-1]["tool_calls"] = followUpRaw.toolCalls
			}
			continue
		}

		if followUpResp.Reply != "" {
			resp.Answer = followUpResp.Reply
		}
		break
	}

	return resp, nil
}

func (p *Provider) callNative(ctx context.Context, req map[string]any) (*ai.Response, *rawChatMessage, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + p.chatPath()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.opts.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)
	}

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API error (%s): %s", httpResp.Status, string(respBody))
	}

	var chatResp struct {
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				Function struct {
					Name      string `json:"name"`
					Arguments any    `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		Done            bool `json:"done"`
		PromptEvalCount int  `json:"prompt_eval_count"`
		EvalCount       int  `json:"eval_count"`
	}

	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	response := &ai.Response{
		Reply: chatResp.Message.Content,
		Usage: ai.Usage{
			InputTokens:  chatResp.PromptEvalCount,
			OutputTokens: chatResp.EvalCount,
			TotalTokens:  chatResp.PromptEvalCount + chatResp.EvalCount,
		},
	}

	var rawToolCalls []map[string]any
	for _, tc := range chatResp.Message.ToolCalls {
		var input map[string]any
		switch v := tc.Function.Arguments.(type) {
		case string:
			if err := json.Unmarshal([]byte(v), &input); err != nil {
				input = map[string]any{}
			}
		case map[string]any:
			input = v
		default:
			input = map[string]any{}
		}
		response.ToolCalls = append(response.ToolCalls, ai.ToolCall{
			Name:  tc.Function.Name,
			Input: input,
		})
		rawToolCalls = append(rawToolCalls, map[string]any{
			"function": map[string]any{
				"name":      tc.Function.Name,
				"arguments": tc.Function.Arguments,
			},
		})
	}

	raw := &rawChatMessage{
		content:   chatResp.Message.Content,
		toolCalls: rawToolCalls,
	}
	return response, raw, nil
}

func (p *Provider) streamNative(ctx context.Context, req *ai.Request) (ai.Stream, error) {
	messages := []map[string]any{}
	if req.SystemPrompt != "" {
		messages = append(messages, map[string]any{"role": "system", "content": req.SystemPrompt})
	}
	for _, m := range req.Messages {
		messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
	}
	if req.Prompt != "" {
		messages = append(messages, map[string]any{"role": "user", "content": req.Prompt})
	}

	apiReq := map[string]any{
		"model":    p.opts.Model,
		"messages": messages,
		"stream":   true,
	}
	if p.opts.MaxTokens > 0 {
		apiReq["options"] = map[string]any{"num_predict": p.opts.MaxTokens}
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stream request: %w", err)
	}

	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + p.streamPath()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.opts.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)
	}

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream API request failed: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("stream API error (%s): %s", httpResp.Status, string(respBody))
	}

	return &ndjsonStream{body: httpResp.Body, scanner: bufio.NewScanner(httpResp.Body)}, nil
}

// ndjsonStream reads newline-delimited JSON (used by local Ollama).
type ndjsonStream struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	closed  bool
}

func (s *ndjsonStream) Recv() (*ai.Response, error) {
	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())
		if line == "" {
			continue
		}
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return nil, fmt.Errorf("failed to parse stream chunk: %w", err)
		}
		if chunk.Done {
			return nil, io.EOF
		}
		if chunk.Message.Content != "" {
			return &ai.Response{Reply: chunk.Message.Content}, nil
		}
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (s *ndjsonStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.body.Close()
}

// rawChatMessage holds the raw assistant content and tool calls for
// follow-up messages.
type rawChatMessage struct {
	content   string
	toolCalls []map[string]any
}
