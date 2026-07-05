// Package anthropic implements the Anthropic Claude model provider
package anthropic

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
	ai.Register("anthropic", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
	ai.RegisterStream("anthropic")
}

// Provider implements the ai.Model interface for Anthropic Claude
type Provider struct {
	opts ai.Options
}

// NewProvider creates a new Anthropic provider
func NewProvider(opts ...ai.Option) *Provider {
	options := ai.NewOptions(opts...)

	// Set defaults if not provided
	if options.Model == "" {
		options.Model = "claude-sonnet-4-20250514"
	}
	if options.BaseURL == "" {
		options.BaseURL = "https://api.anthropic.com"
	}

	return &Provider{
		opts: options,
	}
}

// Init initializes the provider with options
func (p *Provider) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&p.opts)
	}
	return nil
}

// Options returns the provider options
func (p *Provider) Options() ai.Options {
	return p.opts
}

// String returns the provider name
func (p *Provider) String() string {
	return "anthropic"
}

// Generate generates a response from the model
func (p *Provider) Generate(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (*ai.Response, error) {
	// Build tools for Anthropic format
	var anthropicTools []map[string]any
	for _, t := range req.Tools {
		anthropicTools = append(anthropicTools, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"input_schema": map[string]any{
				"type":       "object",
				"properties": t.Properties,
			},
		})
	}

	// Build initial request
	apiReq := map[string]any{
		"model":      p.opts.Model,
		"max_tokens": anthropicMaxTokens(p.opts),
		"system":     req.SystemPrompt,
		"messages":   threadAnthropicMessages(req),
	}

	if len(anthropicTools) > 0 {
		apiReq["tools"] = anthropicTools
	}

	// Make API call
	resp, rawContent, err := p.callAPI(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	// If no tool calls or no handler, return as-is
	if len(resp.ToolCalls) == 0 || p.opts.ToolHandler == nil {
		return resp, nil
	}

	// Tool execution loop: execute tools, send results back, repeat
	// until the model responds with text only (no more tool calls)
	messages := append(threadAnthropicMessages(req),
		map[string]any{"role": "assistant", "content": cleanContent(rawContent)},
	)

	pendingCalls := resp.ToolCalls

	for rounds := 0; rounds < 10; rounds++ {
		var toolResultBlocks []map[string]any
		for i := range pendingCalls {
			content := p.opts.ToolHandler(ctx, pendingCalls[i]).Content
			pendingCalls[i].Result = content
			toolResultBlocks = append(toolResultBlocks, map[string]any{
				"type":        "tool_result",
				"tool_use_id": pendingCalls[i].ID,
				"content":     content,
			})
		}

		messages = append(messages, map[string]any{
			"role":    "user",
			"content": toolResultBlocks,
		})

		followUpReq := map[string]any{
			"model":      p.opts.Model,
			"max_tokens": anthropicMaxTokens(p.opts),
			"system":     req.SystemPrompt,
			"messages":   messages,
		}
		if len(anthropicTools) > 0 {
			followUpReq["tools"] = anthropicTools
		}

		followUpResp, followUpRaw, err := p.callAPI(ctx, followUpReq)
		if err != nil {
			break
		}

		if len(followUpResp.ToolCalls) > 0 {
			resp.ToolCalls = append(resp.ToolCalls, followUpResp.ToolCalls...)
			pendingCalls = followUpResp.ToolCalls
			messages = append(messages, map[string]any{
				"role":    "assistant",
				"content": cleanContent(followUpRaw),
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

// Stream generates a streaming response from Anthropic's Messages SSE API.
func (p *Provider) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
	apiReq := map[string]any{
		"model":      p.opts.Model,
		"max_tokens": anthropicMaxTokens(p.opts),
		"system":     req.SystemPrompt,
		"messages":   threadAnthropicMessages(req),
		"stream":     true,
	}
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stream request: %w", err)
	}

	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-api-key", p.opts.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream API request failed: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("stream API error (%s): %s", httpResp.Status, string(respBody))
	}
	return &streamReader{body: httpResp.Body, scanner: bufio.NewScanner(httpResp.Body)}, nil
}

type streamReader struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	closed  bool
}

func (s *streamReader) Recv() (*ai.Response, error) {
	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") || strings.HasPrefix(line, "event:") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var chunk struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Message struct {
				Usage struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message"`
			Usage *struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("failed to parse stream chunk: %w", err)
		}
		switch chunk.Type {
		case "content_block_delta":
			if chunk.Delta.Type == "text_delta" && chunk.Delta.Text != "" {
				return &ai.Response{Reply: chunk.Delta.Text}, nil
			}
		case "message_start":
			if chunk.Message.Usage.InputTokens > 0 || chunk.Message.Usage.OutputTokens > 0 {
				return &ai.Response{Usage: usage(chunk.Message.Usage.InputTokens, chunk.Message.Usage.OutputTokens)}, nil
			}
		case "message_delta":
			if chunk.Usage != nil {
				return &ai.Response{Usage: usage(chunk.Usage.InputTokens, chunk.Usage.OutputTokens)}, nil
			}
		case "message_stop":
			return nil, io.EOF
		case "error":
			return nil, fmt.Errorf("anthropic stream error: %s", data)
		}
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (s *streamReader) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.body.Close()
}

func usage(input, output int) ai.Usage {
	return ai.Usage{InputTokens: input, OutputTokens: output, TotalTokens: input + output}
}

// callAPI makes an HTTP request to the Anthropic API
func (p *Provider) callAPI(ctx context.Context, req map[string]any) (*ai.Response, any, error) {
	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build HTTP request
	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.opts.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Make request
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response
	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API error (%s): %s", httpResp.Status, string(respBody))
	}

	// Parse response
	var anthropicResp struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}

	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	response := &ai.Response{}

	// Extract text reply
	var replyParts []string
	for _, block := range anthropicResp.Content {
		if block.Type == "text" && block.Text != "" {
			replyParts = append(replyParts, block.Text)
		}
	}
	if len(replyParts) > 0 {
		response.Reply = strings.Join(replyParts, "\n")
	}

	// Extract tool calls
	for _, block := range anthropicResp.Content {
		if block.Type == "tool_use" {
			var input map[string]any
			if err := json.Unmarshal(block.Input, &input); err != nil {
				input = map[string]any{}
			}
			response.ToolCalls = append(response.ToolCalls, ai.ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: input,
			})
		}
	}

	return response, anthropicResp.Content, nil
}

// cleanContent strips fields from response content blocks that Anthropic
// rejects when sent back as assistant message content (e.g. "id" on text blocks).
func cleanContent(raw any) any {
	blocks, ok := raw.([]struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	})
	if !ok {
		return raw
	}
	var cleaned []map[string]any
	for _, b := range blocks {
		switch b.Type {
		case "text":
			cleaned = append(cleaned, map[string]any{"type": "text", "text": b.Text})
		case "tool_use":
			var input any
			_ = json.Unmarshal(b.Input, &input)
			cleaned = append(cleaned, map[string]any{"type": "tool_use", "id": b.ID, "name": b.Name, "input": input})
		}
	}
	return cleaned
}

// threadAnthropicMessages builds the Anthropic messages array from the
// conversation history (req.Messages) followed by the current prompt. The
// system prompt is sent separately via the top-level "system" field.
func threadAnthropicMessages(req *ai.Request) []map[string]any {
	msgs := make([]map[string]any, 0, len(req.Messages)+1)
	for _, m := range req.Messages {
		msgs = append(msgs, map[string]any{"role": m.Role, "content": m.Content})
	}
	if req.Prompt != "" {
		msgs = append(msgs, map[string]any{"role": "user", "content": req.Prompt})
	}
	return msgs
}

func anthropicMaxTokens(o ai.Options) int {
	if o.MaxTokens > 0 {
		return o.MaxTokens
	}
	return 8192
}
