// Package openrouter implements OpenRouter's chat completions API.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/model/internal/catalog"
	"go-micro.dev/v6/model/internal/openaiapi"
)

func init() {
	model.Register("openrouter", func(opts ...model.Option) model.Model { return NewProvider(opts...) })
	model.RegisterStream("openrouter")
}

type Provider struct{ opts model.Options }

func NewProvider(opts ...model.Option) *Provider {
	options := model.NewOptions(opts...)
	if options.Model == "" {
		options.Model = "openrouter/auto"
	}
	if options.BaseURL == "" {
		options.BaseURL = "https://openrouter.ai/api"
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
func (p *Provider) String() string         { return "openrouter" }

// Generate uses complete messages to preserve reasoning metadata between tool calls.
// Token callbacks are buffered until completion; tool streaming is not advertised.
func (p *Provider) Generate(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (*model.Response, error) {
	response, err := openaiapi.Generate(ctx, p.opts, req, p.callAPI)
	if err == nil {
		if onToken := model.NewGenerateOptions(opts...).OnToken; onToken != nil {
			text := response.Answer
			if text == "" {
				text = response.Reply
			}
			onToken(text)
		}
	}
	return response, err
}
func (p *Provider) Stream(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (model.Stream, error) {
	switch p.opts.Model {
	case "minimax/minimax-m3.1-flash:free", "minimax/minimax-m3":
		return openaiapi.StreamWithRequest(ctx, p.opts, req, "/v1/chat/completions", prepareReasoning)
	default:
		return openaiapi.Stream(ctx, p.opts, req, "/v1/chat/completions")
	}
}
func (p *Provider) Models(ctx context.Context) ([]string, error) {
	return catalog.List(ctx, strings.TrimRight(p.opts.BaseURL, "/")+"/v1/models", p.opts.APIKey, "openai")
}

// prepareReasoning uses the gateway's reasoning object.
func prepareReasoning(req map[string]any) {
	if effort, ok := req["reasoning_effort"]; ok {
		req["reasoning"] = map[string]any{"effort": effort}
		delete(req, "reasoning_effort")
	}
}

// callAPI makes one OpenRouter chat completion request.
func (p *Provider) callAPI(ctx context.Context, req map[string]any) (*model.Response, map[string]any, error) {
	prepareReasoning(req)
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
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, nil, err
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, nil, model.NewHTTPError(httpResp, respBody)
	}

	// Parse response
	var chatResp struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
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
	if choice.FinishReason == "length" && strings.TrimSpace(choice.Message.Content) == "" && len(choice.Message.ToolCalls) == 0 {
		return nil, nil, model.ErrOutputLimit
	}
	response := &model.Response{
		Reply: choice.Message.Content,
		Usage: model.Usage{InputTokens: chatResp.Usage.PromptTokens, OutputTokens: chatResp.Usage.CompletionTokens, TotalTokens: chatResp.Usage.TotalTokens},
	}

	response.StopReason = choice.FinishReason

	// Extract tool calls
	for _, tc := range choice.Message.ToolCalls {
		var input map[string]any
		if args := strings.TrimSpace(tc.Function.Arguments); args != "" {
			if err := json.Unmarshal([]byte(args), &input); err != nil {
				return nil, nil, fmt.Errorf("tool %s arguments: %w", tc.Function.Name, err)
			}
		}
		response.ToolCalls = append(response.ToolCalls, model.ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: input,
		})
	}

	// Return raw message for potential follow-up
	var raw struct {
		Choices []struct {
			Message map[string]any `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, nil, err
	}
	rawMessage := raw.Choices[0].Message
	if calls, ok := rawMessage["tool_calls"].([]any); ok {
		for _, call := range calls {
			tool, ok := call.(map[string]any)
			if !ok {
				continue
			}
			function, ok := tool["function"].(map[string]any)
			if !ok {
				continue
			}
			if args, _ := function["arguments"].(string); strings.TrimSpace(args) == "" {
				function["arguments"] = "{}"
			}
		}
	}

	return response, rawMessage, nil
}
