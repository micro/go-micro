// Package minimax implements the MiniMax model provider.
//
// MiniMax offers its flagship MiniMax-M3 model via an OpenAI-compatible
// chat completions endpoint.
//
// Usage:
//
//	import _ "go-micro.dev/v6/model/minimax"
//
//	m := model.New("minimax",
//	    model.WithAPIKey("your-api-key"),
//	)
package minimax

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/model/internal/openaiapi"
)

func init() {
	model.Register("minimax", func(opts ...model.Option) model.Model {
		return NewProvider(opts...)
	})
	model.RegisterStream("minimax")
	model.RegisterToolStream("minimax")
}

type Provider struct {
	opts model.Options
}

func NewProvider(opts ...model.Option) *Provider {
	options := model.NewOptions(opts...)
	if options.Model == "" {
		options.Model = "MiniMax-M3"
	}
	if options.BaseURL == "" {
		options.BaseURL = "https://api.minimax.io"
	}
	return &Provider{opts: options}
}

func (p *Provider) Init(opts ...model.Option) error {
	for _, o := range opts {
		o(&p.opts)
	}
	return nil
}

func (p *Provider) Options() model.Options { return p.opts }
func (p *Provider) String() string         { return "minimax" }

func (p *Provider) Generate(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (*model.Response, error) {
	return openaiapi.Generate(ctx, p.opts, req, p.callAPI)
}

func (p *Provider) Stream(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (model.Stream, error) {
	return openaiapi.Stream(ctx, p.opts, req, "/v1/chat/completions")
}

func (p *Provider) callAPI(ctx context.Context, req map[string]any) (*model.Response, map[string]any, error) {
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
		return nil, nil, model.NewHTTPError(httpResp, respBody)
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
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
	response := &model.Response{Reply: choice.Message.Content}

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

	rawMessage := map[string]any{
		"content":    choice.Message.Content,
		"tool_calls": choice.Message.ToolCalls,
	}

	return response, rawMessage, nil
}
