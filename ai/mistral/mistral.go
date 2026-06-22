// Package mistral implements the Mistral AI model provider.
//
// Mistral AI is a European AI company offering high-performance models
// via an OpenAI-compatible chat completions endpoint.
//
// Usage:
//
//	import _ "go-micro.dev/v6/ai/mistral"
//
//	m := ai.New("mistral",
//	    ai.WithAPIKey("your-api-key"),
//	)
package mistral

import (
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
	ai.Register("mistral", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
}

type Provider struct {
	opts ai.Options
}

func NewProvider(opts ...ai.Option) *Provider {
	options := ai.NewOptions(opts...)
	if options.Model == "" {
		options.Model = "mistral-large-latest"
	}
	if options.BaseURL == "" {
		options.BaseURL = "https://api.mistral.ai"
	}
	return &Provider{opts: options}
}

func (p *Provider) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&p.opts)
	}
	return nil
}

func (p *Provider) Options() ai.Options { return p.opts }
func (p *Provider) String() string      { return "mistral" }

func (p *Provider) Generate(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (*ai.Response, error) {
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

	messages := []map[string]any{
		{"role": "system", "content": req.SystemPrompt},
		{"role": "user", "content": req.Prompt},
	}

	apiReq := map[string]any{
		"model":    p.opts.Model,
		"messages": messages,
	}
	if len(tools) > 0 {
		apiReq["tools"] = tools
	}

	resp, rawMessage, err := p.callAPI(ctx, apiReq)
	if err != nil {
		return nil, err
	}
	if len(resp.ToolCalls) == 0 {
		return resp, nil
	}

	if p.opts.ToolHandler != nil {
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
		followUpResp, _, err := p.callAPI(ctx, map[string]any{
			"model":    p.opts.Model,
			"messages": followUpMessages,
		})
		if err == nil && followUpResp.Reply != "" {
			resp.Answer = followUpResp.Reply
		}
	}

	return resp, nil
}

func (p *Provider) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
	return nil, fmt.Errorf("streaming not yet implemented for mistral provider")
}

func (p *Provider) callAPI(ctx context.Context, req map[string]any) (*ai.Response, map[string]any, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)

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
	response := &ai.Response{Reply: choice.Message.Content}

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

	rawMessage := map[string]any{
		"content":    choice.Message.Content,
		"tool_calls": choice.Message.ToolCalls,
	}

	return response, rawMessage, nil
}
