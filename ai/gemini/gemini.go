// Package gemini implements the Google Gemini model provider.
//
// Usage:
//
//	import _ "go-micro.dev/v6/ai/gemini"
//
//	m := ai.New("gemini",
//	    ai.WithAPIKey("your-api-key"),
//	)
package gemini

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
	ai.Register("gemini", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
}

// Provider implements the ai.Model interface for Google Gemini.
type Provider struct {
	opts ai.Options
}

// NewProvider creates a new Gemini provider.
func NewProvider(opts ...ai.Option) *Provider {
	options := ai.NewOptions(opts...)

	if options.Model == "" {
		options.Model = "gemini-2.5-flash"
	}
	if options.BaseURL == "" {
		options.BaseURL = "https://generativelanguage.googleapis.com"
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
func (p *Provider) String() string      { return "gemini" }

func (p *Provider) Generate(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (*ai.Response, error) {
	var tools []map[string]any
	for _, t := range req.Tools {
		tools = append(tools, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"parameters": map[string]any{
				"type":       "object",
				"properties": t.Properties,
			},
		})
	}

	contents := []map[string]any{
		{"role": "user", "parts": []map[string]any{{"text": req.Prompt}}},
	}

	apiReq := map[string]any{
		"contents": contents,
	}

	if req.SystemPrompt != "" {
		apiReq["system_instruction"] = map[string]any{
			"parts": []map[string]any{{"text": req.SystemPrompt}},
		}
	}

	if len(tools) > 0 {
		apiReq["tools"] = []map[string]any{
			{"functionDeclarations": tools},
		}
	}

	resp, rawParts, err := p.callAPI(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	if len(resp.ToolCalls) == 0 {
		return resp, nil
	}

	if p.opts.ToolHandler != nil {
		var resultParts []map[string]any
		for _, tc := range resp.ToolCalls {
			result := p.opts.ToolHandler(ctx, tc).Value
			resultParts = append(resultParts, map[string]any{
				"functionResponse": map[string]any{
					"name":     tc.Name,
					"id":       tc.ID,
					"response": result,
				},
			})
		}

		followUpContents := append(contents,
			map[string]any{"role": "model", "parts": rawParts},
			map[string]any{"role": "user", "parts": resultParts},
		)

		followUpReq := map[string]any{
			"contents": followUpContents,
		}
		if req.SystemPrompt != "" {
			followUpReq["system_instruction"] = map[string]any{
				"parts": []map[string]any{{"text": req.SystemPrompt}},
			}
		}

		followUpResp, _, err := p.callAPI(ctx, followUpReq)
		if err == nil && followUpResp.Reply != "" {
			resp.Answer = followUpResp.Reply
		}
	}

	return resp, nil
}

func (p *Provider) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
	return nil, fmt.Errorf("%w: gemini provider", ai.ErrStreamingUnsupported)
}

func (p *Provider) callAPI(ctx context.Context, req map[string]any) (*ai.Response, []map[string]any, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := strings.TrimRight(p.opts.BaseURL, "/") +
		"/v1beta/models/" + p.opts.Model + ":generateContent"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.opts.APIKey)

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API error (%s): %s", httpResp.Status, string(respBody))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string          `json:"text"`
					FunctionCall *functionCallPB `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, nil, fmt.Errorf("no response from API")
	}

	parts := geminiResp.Candidates[0].Content.Parts
	response := &ai.Response{}

	var replyParts []string
	var rawParts []map[string]any

	for _, part := range parts {
		if part.Text != "" {
			replyParts = append(replyParts, part.Text)
			rawParts = append(rawParts, map[string]any{"text": part.Text})
		}
		if part.FunctionCall != nil {
			response.ToolCalls = append(response.ToolCalls, ai.ToolCall{
				ID:    part.FunctionCall.ID,
				Name:  part.FunctionCall.Name,
				Input: part.FunctionCall.Args,
			})
			rawParts = append(rawParts, map[string]any{
				"functionCall": map[string]any{
					"id":   part.FunctionCall.ID,
					"name": part.FunctionCall.Name,
					"args": part.FunctionCall.Args,
				},
			})
		}
	}

	if len(replyParts) > 0 {
		response.Reply = strings.Join(replyParts, "\n")
	}

	return response, rawParts, nil
}

type functionCallPB struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}
