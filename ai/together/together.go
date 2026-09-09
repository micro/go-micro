// Package together implements the Together AI model provider.
//
// Together AI provides fast inference for open-weight models via an
// OpenAI-compatible chat completions endpoint.
//
// Usage:
//
//	import _ "go-micro.dev/v6/ai/together"
//
//	m := ai.New("together",
//	    ai.WithAPIKey("your-api-key"),
//	)
package together

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/ai/internal/openaiapi"
)

func init() {
	ai.Register("together", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
	ai.RegisterStream("together")
	ai.RegisterToolStream("together")
}

type Provider struct {
	opts ai.Options
}

func NewProvider(opts ...ai.Option) *Provider {
	options := ai.NewOptions(opts...)
	if options.Model == "" {
		options.Model = "meta-llama/Llama-3.3-70B-Instruct-Turbo"
	}
	if options.BaseURL == "" {
		options.BaseURL = "https://api.together.xyz"
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
func (p *Provider) String() string      { return "together" }

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

	// Tool execution loop: execute tools, send results back, and keep the
	// tools on offer so the model can take the next step. A follow-up without
	// "tools" asks the model to continue with its hands tied — the call it
	// wanted comes back written out as prose — and without a loop a second
	// step is impossible whatever the model wants. Bounded so a model that
	// never stops asking cannot run forever.
	if p.opts.ToolHandler != nil {
		// Copied rather than aliased: append on a slice that shares an array
		// with messages would overwrite it on a later round.
		followUpMessages := append([]map[string]any(nil), messages...)
		pending := resp.ToolCalls
		raw := rawMessage
		for round := 0; len(pending) > 0 && round < maxToolRounds; round++ {
			followUpMessages = append(followUpMessages, map[string]any{
				"role":       "assistant",
				"content":    raw["content"],
				"tool_calls": raw["tool_calls"],
			})
			for _, tc := range pending {
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
			if len(tools) > 0 {
				followUpReq["tools"] = tools
			}

			followUpResp, followUpRaw, err := p.callAPI(ctx, followUpReq)
			if err != nil {
				break
			}
			if followUpResp.Reply != "" {
				resp.Answer = followUpResp.Reply
			}
			pending, raw = followUpResp.ToolCalls, followUpRaw
			resp.ToolCalls = append(resp.ToolCalls, followUpResp.ToolCalls...)
		}
	}

	return resp, nil
}

// maxToolRounds bounds the tool-execution loop in a single Generate. Each
// round is a model call plus the tools it asks for, so this is the ceiling on
// one question's cost as well as its length; it is high enough that no honest
// piece of multi-step work reaches it.
const maxToolRounds = 12

func (p *Provider) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
	return openaiapi.Stream(ctx, p.opts, req, "/v1/chat/completions")
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
		return nil, nil, ai.NewHTTPError(httpResp, respBody)
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
