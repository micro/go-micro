// Package together implements the Together AI model provider.
//
// Together AI provides fast inference for open-weight models via an
// OpenAI-compatible chat completions endpoint.
//
// Usage:
//
//	import _ "go-micro.dev/v6/model/together"
//
//	m := model.New("together",
//	    model.WithAPIKey("your-api-key"),
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

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/model/internal/openaiapi"
)

func init() {
	model.Register("together", func(opts ...model.Option) model.Model {
		return NewProvider(opts...)
	})
	model.RegisterStream("together")
	model.RegisterToolStream("together")
}

type Provider struct {
	opts model.Options
}

func NewProvider(opts ...model.Option) *Provider {
	options := model.NewOptions(opts...)
	if options.Model == "" {
		options.Model = "meta-llama/Llama-3.3-70B-Instruct-Turbo"
	}
	if options.BaseURL == "" {
		options.BaseURL = "https://api.together.xyz"
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
func (p *Provider) String() string         { return "together" }

func (p *Provider) Generate(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (*model.Response, error) {
	options := p.opts
	if model.IsSingleTurn(opts) {
		options.ToolHandler = nil
	}
	return openaiapi.Generate(ctx, options, req, p.callAPI)
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

	var raw struct {
		Choices []struct {
			Message      map[string]any `json:"message"`
			FinishReason string         `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			Input  int `json:"prompt_tokens"`
			Output int `json:"completion_tokens"`
			Total  int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, nil, err
	}
	rawMessage := raw.Choices[0].Message
	rawMessage["role"] = "assistant"
	response.StopReason = raw.Choices[0].FinishReason
	response.Usage = model.Usage{InputTokens: raw.Usage.Input, OutputTokens: raw.Usage.Output, TotalTokens: raw.Usage.Total}

	return response, rawMessage, nil
}

func (p *Provider) Turn(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (*model.Response, error) {
	return model.SingleTurn(ctx, p, req, opts...)
}
