// Package openai implements the OpenAI model provider
package openai

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
	ai.Register("openai", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
	ai.RegisterImage("openai", func(opts ...ai.Option) ai.ImageModel {
		return NewProvider(opts...)
	})
	ai.RegisterStream("openai")
	ai.RegisterToolStream("openai")
}

// Provider implements the ai.Model interface for OpenAI
type Provider struct {
	opts ai.Options
}

// NewProvider creates a new OpenAI provider
func NewProvider(opts ...ai.Option) *Provider {
	options := ai.NewOptions(opts...)

	// Set defaults if not provided
	if options.Model == "" {
		options.Model = "gpt-4o"
	}
	if options.BaseURL == "" {
		options.BaseURL = "https://api.openai.com"
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
	return "openai"
}

// Generate generates a response from the model
func (p *Provider) Generate(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (*ai.Response, error) {
	// Build tools for OpenAI format
	var openaiTools []map[string]any
	for _, t := range req.Tools {
		openaiTools = append(openaiTools, map[string]any{
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

	// Build messages
	messages := []map[string]any{
		{"role": "system", "content": req.SystemPrompt},
	}
	for _, m := range req.Messages {
		messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
	}
	if req.Prompt != "" {
		messages = append(messages, map[string]any{"role": "user", "content": req.Prompt})
	}

	// Build initial request
	apiReq := map[string]any{
		"model":    p.opts.Model,
		"messages": messages,
	}
	if p.opts.MaxTokens > 0 {
		apiReq["max_tokens"] = p.opts.MaxTokens
	}

	if len(openaiTools) > 0 {
		apiReq["tools"] = openaiTools
	}

	// Make API call
	resp, rawMessage, err := p.callAPI(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	// If no tool calls, return response
	if len(resp.ToolCalls) == 0 {
		return resp, nil
	}

	// If tool handler is provided, execute tools and get final answer
	if p.opts.ToolHandler != nil {
		// Build follow-up messages
		followUpMessages := append(messages, map[string]any{
			"role":       "assistant",
			"content":    rawMessage["content"],
			"tool_calls": rawMessage["tool_calls"],
		})

		for _, tc := range resp.ToolCalls {
			content := p.opts.ToolHandler(ctx, tc).Content
			followUpMessages = append(followUpMessages, map[string]any{
				"role":         "tool",
				"tool_call_id": tc.ID,
				"content":      content,
			})
		}

		followUpReq := map[string]any{
			"model":    p.opts.Model,
			"messages": followUpMessages,
		}

		// Make follow-up API call
		followUpResp, _, err := p.callAPI(ctx, followUpReq)
		if err == nil && followUpResp.Reply != "" {
			resp.Answer = followUpResp.Reply
		}
	}

	return resp, nil
}

// Stream generates a streaming response from the OpenAI chat completions API.
func (p *Provider) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
	messages := []map[string]any{
		{"role": "system", "content": req.SystemPrompt},
	}
	for _, m := range req.Messages {
		messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
	}
	if req.Prompt != "" {
		messages = append(messages, map[string]any{"role": "user", "content": req.Prompt})
	}
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
	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream API request failed: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("stream API error (%s): %s", httpResp.Status, string(respBody))
	}
	return &openAIStream{body: httpResp.Body, scanner: bufio.NewScanner(httpResp.Body)}, nil
}

type openAIStream struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	closed  bool
}

func (s *openAIStream) Recv() (*ai.Response, error) {
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
		// Final chunk (after include_usage) carries token usage and no content.
		if chunk.Usage != nil {
			return &ai.Response{Usage: ai.Usage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
				TotalTokens:  chunk.Usage.TotalTokens,
			}}, nil
		}
		continue
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (s *openAIStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.body.Close()
}

// callAPI makes an HTTP request to the OpenAI API
func (p *Provider) callAPI(ctx context.Context, req map[string]any) (*ai.Response, map[string]any, error) {
	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build HTTP request
	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)

	// Make request
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response
	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return nil, nil, ai.NewHTTPError(httpResp, respBody)
	}

	// Parse response
	var chatResp struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Choices []struct {
			Message struct {
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
		Usage: ai.Usage{InputTokens: chatResp.Usage.PromptTokens, OutputTokens: chatResp.Usage.CompletionTokens, TotalTokens: chatResp.Usage.TotalTokens},
	}

	// Extract tool calls
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
	}

	// Return raw message for potential follow-up
	rawMessage := map[string]any{
		"content":    choice.Message.Content,
		"tool_calls": choice.Message.ToolCalls,
	}

	return response, rawMessage, nil
}

const defaultImageModel = "gpt-image-1"

func (p *Provider) GenerateImage(ctx context.Context, req *ai.ImageRequest, opts ...ai.GenerateOption) (*ai.ImageResponse, error) {
	model := req.Model
	if model == "" {
		model = defaultImageModel
	}
	n := req.N
	if n <= 0 {
		n = 1
	}

	apiReq := map[string]any{
		"model":  model,
		"prompt": req.Prompt,
		"n":      n,
	}
	if req.Size != "" {
		apiReq["size"] = req.Size
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/images/generations"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%s): %s", httpResp.Status, string(respBody))
	}

	var imgResp struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &imgResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	response := &ai.ImageResponse{}
	for _, d := range imgResp.Data {
		response.Images = append(response.Images, ai.Image{
			URL:    d.URL,
			Base64: d.B64JSON,
		})
	}

	return response, nil
}
