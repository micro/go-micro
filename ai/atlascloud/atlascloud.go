// Package atlascloud implements the Atlas Cloud model provider.
//
// Atlas Cloud is an enterprise AI infrastructure platform offering
// high-performance LLM, image, and video APIs. It exposes
// OpenAI-compatible endpoints for chat completions and image
// generation.
//
// Usage:
//
//	import _ "go-micro.dev/v5/ai/atlascloud"
//
//	m := ai.New("atlascloud",
//	    ai.WithAPIKey("your-api-key"),
//	)
//
//	// Image generation
//	ig := ai.NewImage("atlascloud",
//	    ai.WithAPIKey("your-api-key"),
//	)
package atlascloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go-micro.dev/v5/ai"
)

func init() {
	ai.Register("atlascloud", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
	ai.RegisterImage("atlascloud", func(opts ...ai.Option) ai.ImageModel {
		return NewProvider(opts...)
	})
}

// Provider implements the ai.Model interface for Atlas Cloud.
type Provider struct {
	opts ai.Options
}

// NewProvider creates a new Atlas Cloud provider.
func NewProvider(opts ...ai.Option) *Provider {
	options := ai.NewOptions(opts...)

	if options.Model == "" {
		options.Model = "deepseek-ai/DeepSeek-V3-0324"
	}
	if options.BaseURL == "" {
		options.BaseURL = "https://api.atlascloud.ai"
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
func (p *Provider) String() string      { return "atlascloud" }

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

		followUpResp, _, err := p.callAPI(ctx, followUpReq)
		if err == nil && followUpResp.Reply != "" {
			resp.Answer = followUpResp.Reply
		}
	}

	return resp, nil
}

func (p *Provider) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
	return nil, fmt.Errorf("streaming not yet implemented for atlascloud provider")
}

func (p *Provider) callAPI(ctx context.Context, req map[string]any) (*ai.Response, map[string]any, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(reqBody))
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
	if httpResp.StatusCode != 200 {
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
	response := &ai.Response{
		Reply: choice.Message.Content,
	}

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

const defaultImageModel = "openai/gpt-image-2/text-to-image"

// GenerateImage creates an image using Atlas Cloud's async image API.
// It submits the job and polls until completion or context cancellation.
func (p *Provider) GenerateImage(ctx context.Context, req *ai.ImageRequest, opts ...ai.GenerateOption) (*ai.ImageResponse, error) {
	model := req.Model
	if model == "" {
		model = defaultImageModel
	}
	quality := req.Quality
	if quality == "" {
		quality = "medium"
	}
	outputFmt := req.OutputFormat
	if outputFmt == "" {
		outputFmt = "png"
	}
	size := req.Size
	if size == "" {
		size = "1024x1024"
	}

	apiReq := map[string]any{
		"model":                model,
		"prompt":               req.Prompt,
		"quality":              quality,
		"output_format":        outputFmt,
		"size":                 size,
		"enable_sync_mode":     false,
		"enable_base64_output": false,
		"moderation":           "low",
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/api/v1/model/generateImage"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(reqBody))
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
	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (%s): %s", httpResp.Status, string(respBody))
	}

	var submitResp struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &submitResp); err != nil {
		return nil, fmt.Errorf("failed to parse submit response: %w", err)
	}
	if submitResp.Code != 200 {
		return nil, fmt.Errorf("API error: %s", submitResp.Msg)
	}

	predictionID := submitResp.Data.ID
	pollURL := strings.TrimRight(p.opts.BaseURL, "/") + "/api/v1/model/prediction/" + predictionID

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			result, err := p.pollPrediction(ctx, pollURL)
			if err != nil {
				return nil, err
			}
			if result != nil {
				return result, nil
			}
		}
	}
}

func (p *Provider) pollPrediction(ctx context.Context, url string) (*ai.ImageResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("poll request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)

	var pollResp struct {
		Data struct {
			Status  string   `json:"status"`
			Outputs []string `json:"outputs"`
			Error   string   `json:"error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &pollResp); err != nil {
		return nil, fmt.Errorf("failed to parse poll response: %w", err)
	}

	switch pollResp.Data.Status {
	case "completed":
		resp := &ai.ImageResponse{}
		for _, output := range pollResp.Data.Outputs {
			resp.Images = append(resp.Images, ai.Image{URL: output})
		}
		return resp, nil
	case "failed":
		return nil, fmt.Errorf("image generation failed: %s", pollResp.Data.Error)
	default:
		return nil, nil
	}
}
