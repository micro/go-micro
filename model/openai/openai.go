// Package openai implements the OpenAI model provider
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-micro.dev/v5/model"
)

func init() {
	model.Register("openai", func(opts ...model.Option) model.Model {
		return NewProvider(opts...)
	})
}

// Provider implements the model.Model interface for OpenAI
type Provider struct {
	opts model.Options
}

// NewProvider creates a new OpenAI provider
func NewProvider(opts ...model.Option) *Provider {
	options := model.NewOptions(opts...)
	
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
func (p *Provider) Init(opts ...model.Option) error {
	for _, o := range opts {
		o(&p.opts)
	}
	return nil
}

// Options returns the provider options
func (p *Provider) Options() model.Options {
	return p.opts
}

// String returns the provider name
func (p *Provider) String() string {
	return "openai"
}

// Generate generates a response from the model
func (p *Provider) Generate(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (*model.Response, error) {
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
		{"role": "user", "content": req.Prompt},
	}

	// Build initial request
	apiReq := map[string]any{
		"model":    p.opts.Model,
		"messages": messages,
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
			_, content := p.opts.ToolHandler(tc.Name, tc.Input)
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

// Stream generates a streaming response (not yet implemented)
func (p *Provider) Stream(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (model.Stream, error) {
	return nil, fmt.Errorf("streaming not yet implemented for openai provider")
}

// callAPI makes an HTTP request to the OpenAI API
func (p *Provider) callAPI(ctx context.Context, req map[string]any) (*model.Response, map[string]any, error) {
	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build HTTP request
	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(reqBody))
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
	if httpResp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("API error (%s): %s", httpResp.Status, string(respBody))
	}

	// Parse response
	var chatResp struct {
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
	response := &model.Response{
		Reply: choice.Message.Content,
	}

	// Extract tool calls
	for _, tc := range choice.Message.ToolCalls {
		var input map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
			input = map[string]any{}
		}
		response.ToolCalls = append(response.ToolCalls, model.ToolCall{
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
