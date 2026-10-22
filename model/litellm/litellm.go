// Package litellm implements the LiteLLM proxy model provider.
//
// LiteLLM (https://docs.litellm.ai/docs/simple_proxy) is a self-hosted,
// OpenAI-compatible AI gateway in front of 100+ providers (OpenAI, Anthropic,
// Bedrock, Vertex AI, Azure, Ollama, ...). Model names are the model_name
// aliases configured on the proxy.
//
// The API key is optional: a proxy started without a master key accepts
// unauthenticated requests, and with a master or virtual key the key is sent
// as a bearer token. The provider also implements model.ModelLister (the
// proxy's /v1/models) and model.ImageModel (/v1/images/generations).
//
// Usage:
//
//	import _ "go-micro.dev/v6/model/litellm"
//
//	m := model.New("litellm",
//	    model.WithBaseURL("http://localhost:4000"),
//	    model.WithModel("gpt-4o-mini"),
//	)
package litellm

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

const (
	defaultBaseURL = "http://localhost:4000"
	defaultModel   = "gpt-4o-mini"
)

func init() {
	model.Register("litellm", func(opts ...model.Option) model.Model {
		return NewProvider(opts...)
	})
	model.RegisterImage("litellm", func(opts ...model.Option) model.ImageModel {
		return NewProvider(opts...)
	})
	model.RegisterStream("litellm")
	model.RegisterToolStream("litellm")
}

// Provider talks to a LiteLLM proxy.
type Provider struct {
	opts model.Options
}

// NewProvider creates a LiteLLM provider. The base URL defaults to
// http://localhost:4000 (the provider appends /v1).
func NewProvider(opts ...model.Option) *Provider {
	options := model.NewOptions(opts...)
	if options.Model == "" {
		options.Model = defaultModel
	}
	if options.BaseURL == "" {
		options.BaseURL = defaultBaseURL
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
func (p *Provider) String() string         { return "litellm" }

// Generate runs a chat completion, executing tool calls through the
// configured tool handler.
func (p *Provider) Generate(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (*model.Response, error) {
	return openaiapi.Generate(ctx, p.opts, req, p.callAPI, opts...)
}

// Stream streams a chat completion, including requests with tools.
func (p *Provider) Stream(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (model.Stream, error) {
	return openaiapi.Stream(ctx, p.opts, req, "/v1/chat/completions")
}

// Models lists the model aliases the proxy serves. With a virtual key the
// proxy returns only the models that key may call.
func (p *Provider) Models(ctx context.Context) ([]string, error) {
	return catalog.List(ctx, p.endpoint("/v1/models"), p.opts.APIKey, "openai")
}

// GenerateImage creates images through the proxy's /v1/images/generations,
// which LiteLLM routes to OpenAI, Azure, Vertex AI Imagen, Bedrock, and other
// image providers.
func (p *Provider) GenerateImage(ctx context.Context, req *model.ImageRequest, opts ...model.GenerateOption) (*model.ImageResponse, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("litellm: image prompt is empty")
	}
	imageModel := req.Model
	if imageModel == "" {
		imageModel = p.opts.Model
	}
	body := map[string]any{
		"model":  imageModel,
		"prompt": req.Prompt,
		// Image models that only return base64 (Vertex AI, Gemini) reject
		// unsupported fields; let the proxy drop them instead of failing.
		"drop_params": true,
	}
	if req.N > 0 {
		body["n"] = req.N
	}
	if req.Size != "" {
		body["size"] = req.Size
	}
	if req.Quality != "" {
		body["quality"] = req.Quality
	}
	if req.OutputFormat != "" {
		body["output_format"] = req.OutputFormat
	}

	respBody, err := p.post(ctx, "/v1/images/generations", body)
	if err != nil {
		return nil, err
	}
	var imageResp struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &imageResp); err != nil {
		return nil, fmt.Errorf("failed to parse image response: %w", err)
	}
	result := &model.ImageResponse{}
	for _, image := range imageResp.Data {
		if image.URL == "" && image.B64JSON == "" {
			continue
		}
		result.Images = append(result.Images, model.Image{URL: image.URL, Base64: image.B64JSON})
	}
	if len(result.Images) == 0 {
		return nil, fmt.Errorf("litellm: no image returned for model %q", imageModel)
	}
	return result, nil
}

func (p *Provider) endpoint(path string) string {
	return strings.TrimSuffix(strings.TrimRight(p.opts.BaseURL, "/"), "/v1") + path
}

// post sends a JSON request and returns the body of a 200 response.
func (p *Provider) post(ctx context.Context, path string, body map[string]any) ([]byte, error) {
	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint(path), bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.opts.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)
	}

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, model.NewHTTPError(httpResp, respBody)
	}
	return respBody, nil
}

// callAPI makes one chat completion request.
func (p *Provider) callAPI(ctx context.Context, req map[string]any) (*model.Response, map[string]any, error) {
	respBody, err := p.post(ctx, "/v1/chat/completions", req)
	if err != nil {
		return nil, nil, err
	}

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
		Reply:      choice.Message.Content,
		StopReason: choice.FinishReason,
		Usage: model.Usage{
			InputTokens:  chatResp.Usage.PromptTokens,
			OutputTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:  chatResp.Usage.TotalTokens,
		},
	}
	for _, tc := range choice.Message.ToolCalls {
		var input map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
			return nil, nil, fmt.Errorf("tool %s arguments: %w", tc.Function.Name, err)
		}
		response.ToolCalls = append(response.ToolCalls, model.ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: input,
		})
	}

	// Return the raw message so tool follow-ups keep provider metadata.
	var raw struct {
		Choices []struct {
			Message map[string]any `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, nil, err
	}
	return response, raw.Choices[0].Message, nil
}
