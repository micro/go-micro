// Package anthropic implements the Anthropic Claude model provider
package anthropic

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
	model.Register("anthropic", func(opts ...model.Option) model.Model {
		return NewProvider(opts...)
	})
}

// Provider implements the model.Model interface for Anthropic Claude
type Provider struct {
	opts model.Options
}

// NewProvider creates a new Anthropic provider
func NewProvider(opts ...model.Option) *Provider {
	options := model.NewOptions(opts...)
	
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
	return "anthropic"
}

// Generate generates a response from the model
func (p *Provider) Generate(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (*model.Response, error) {
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
		"max_tokens": 4096,
		"system":     req.SystemPrompt,
		"messages": []map[string]any{
			{"role": "user", "content": req.Prompt},
		},
	}

	if len(anthropicTools) > 0 {
		apiReq["tools"] = anthropicTools
	}

	// Make API call
	resp, rawContent, err := p.callAPI(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	// If no tool calls, return response
	if len(resp.ToolCalls) == 0 {
		return resp, nil
	}

	// If tool handler is provided, execute tools and get final answer
	if p.opts.ToolHandler != nil {
		var toolResults []model.ToolResult
		for _, tc := range resp.ToolCalls {
			_, content := p.opts.ToolHandler(tc.Name, tc.Input)
			toolResults = append(toolResults, model.ToolResult{
				ID:      tc.ID,
				Content: content,
			})
		}

		// Build follow-up request with tool results
		var toolResultBlocks []map[string]any
		for _, tr := range toolResults {
			toolResultBlocks = append(toolResultBlocks, map[string]any{
				"type":        "tool_result",
				"tool_use_id": tr.ID,
				"content":     tr.Content,
			})
		}

		followUpReq := map[string]any{
			"model":      p.opts.Model,
			"max_tokens": 4096,
			"system":     req.SystemPrompt,
			"messages": []map[string]any{
				{"role": "user", "content": req.Prompt},
				{"role": "assistant", "content": rawContent},
				{"role": "user", "content": toolResultBlocks},
			},
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
	return nil, fmt.Errorf("streaming not yet implemented for anthropic provider")
}

// callAPI makes an HTTP request to the Anthropic API
func (p *Provider) callAPI(ctx context.Context, req map[string]any) (*model.Response, any, error) {
	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build HTTP request
	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(reqBody))
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
	if httpResp.StatusCode != 200 {
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

	response := &model.Response{}

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
			response.ToolCalls = append(response.ToolCalls, model.ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: input,
			})
		}
	}

	return response, anthropicResp.Content, nil
}
