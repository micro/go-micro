// Package atlascloud implements the Atlas Cloud model provider.
//
// Atlas Cloud is an enterprise AI infrastructure platform offering
// high-performance LLM, image, and video APIs. It exposes
// OpenAI-compatible endpoints for chat completions and image
// generation.
//
// Usage:
//
//	import _ "go-micro.dev/v6/ai/atlascloud"
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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go-micro.dev/v6/ai"
)

func init() {
	ai.Register("atlascloud", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
	ai.RegisterImage("atlascloud", func(opts ...ai.Option) ai.ImageModel {
		return NewProvider(opts...)
	})
	ai.RegisterVideo("atlascloud", func(opts ...ai.Option) ai.VideoModel {
		return NewProvider(opts...)
	})
	ai.RegisterStream("atlascloud")
}

// Provider implements the ai.Model interface for Atlas Cloud.
type Provider struct {
	opts ai.Options
}

type atlasToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// NewProvider creates a new Atlas Cloud provider.
func NewProvider(opts ...ai.Option) *Provider {
	options := ai.NewOptions(opts...)

	if options.Model == "" {
		// Allow the chat model to be selected via the ATLASCLOUD_MODEL env var
		// (e.g. to run CI conformance against a stronger tool-use model) without
		// a code change; fall back to a sensible default otherwise.
		if m := os.Getenv("ATLASCLOUD_MODEL"); m != "" {
			options.Model = m
		} else {
			options.Model = "deepseek-ai/DeepSeek-V3-0324"
		}
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
	}
	for _, m := range req.Messages {
		messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
	}
	if req.Prompt != "" {
		messages = append(messages, map[string]any{"role": "user", "content": req.Prompt})
	}

	apiReq := map[string]any{
		"model":    p.opts.Model,
		"messages": messages,
	}
	if p.opts.MaxTokens > 0 {
		apiReq["max_tokens"] = p.opts.MaxTokens
	}

	if len(tools) > 0 {
		apiReq["tools"] = tools
	}

	resp, rawMessage, err := p.callAPI(ctx, "chat", apiReq)
	if err != nil {
		return nil, err
	}

	if len(resp.ToolCalls) == 0 {
		return resp, nil
	}

	if p.opts.ToolHandler != nil {
		allToolCalls := append([]ai.ToolCall(nil), resp.ToolCalls...)
		var toolResults []string
		followUpMessages := append(messages, map[string]any{
			"role":       "assistant",
			"content":    rawMessage["content"],
			"tool_calls": rawMessage["tool_calls"],
		})

		for _, tc := range resp.ToolCalls {
			content := p.opts.ToolHandler(ctx, tc).Content
			if content != "" {
				toolResults = append(toolResults, content)
			}
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
			// Keep the tool schema available during the follow-up turn. Minimax
			// models behind Atlas Cloud sometimes call one required tool, inspect
			// that result, and then issue a second tool call (for example a guarded
			// delegate conformance check) instead of completing immediately.
			followUpReq["tools"] = tools
		}

		followUpResp, _, err := p.callAPI(ctx, "tool-follow-up", followUpReq)
		if err != nil {
			return nil, err
		}
		if len(followUpResp.ToolCalls) > 0 {
			for i := range followUpResp.ToolCalls {
				result := p.opts.ToolHandler(ctx, followUpResp.ToolCalls[i])
				if result.Refused != "" {
					followUpResp.ToolCalls[i].Error = result.Refused
				}
				if result.Content != "" {
					followUpResp.ToolCalls[i].Result = result.Content
					toolResults = append(toolResults, result.Content)
				}
			}
			allToolCalls = append(allToolCalls, followUpResp.ToolCalls...)
			resp.ToolCalls = allToolCalls
		}
		if followUpResp.Reply != "" {
			resp.Answer = followUpResp.Reply
		} else if len(toolResults) > 0 {
			resp.Answer = strings.Join(toolResults, "\n")
		}
	}

	return resp, nil
}

// Stream generates a streaming response from Atlas Cloud's OpenAI-compatible
// chat completions endpoint, emitting content deltas as they arrive.
func (p *Provider) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
	if len(req.Tools) > 0 {
		return nil, fmt.Errorf("%w: atlascloud streaming does not expose tools", ai.ErrStreamingUnsupported)
	}

	messages := []map[string]any{
		{"role": "system", "content": req.SystemPrompt},
	}
	for _, m := range req.Messages {
		messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
	}
	if req.Prompt != "" {
		messages = append(messages, map[string]any{"role": "user", "content": req.Prompt})
	}
	apiReq := map[string]any{
		"model":          p.opts.Model,
		"messages":       messages,
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	if p.opts.MaxTokens > 0 {
		apiReq["max_tokens"] = p.opts.MaxTokens
	}
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stream request: %w", err)
	}
	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream API request failed: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("stream API error (%s): %s", httpResp.Status, string(respBody))
	}
	return &atlasStream{body: httpResp.Body, scanner: bufio.NewScanner(httpResp.Body)}, nil
}

type atlasStream struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	closed  bool
}

func (s *atlasStream) Recv() (*ai.Response, error) {
	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil, io.EOF
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("failed to parse stream chunk: %w", err)
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			return &ai.Response{Reply: chunk.Choices[0].Delta.Content}, nil
		}
		// Final chunk (after include_usage) carries token usage and no content.
		if chunk.Usage != nil {
			return &ai.Response{Usage: ai.Usage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
				TotalTokens:  chunk.Usage.TotalTokens,
			}}, nil
		}
		continue
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (s *atlasStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.body.Close()
}

func (p *Provider) callAPI(ctx context.Context, phase string, req map[string]any) (*ai.Response, map[string]any, error) {
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
		return nil, nil, fmt.Errorf("API error (%s) during atlascloud %s request (%s): %s", httpResp.Status, phase, atlascloudRequestSummary(req), string(respBody))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content   string          `json:"content"`
				ToolCalls []atlasToolCall `json:"tool_calls"`
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
		"tool_calls": normalizeAtlasCloudToolCalls(choice.Message.ToolCalls),
	}

	return response, rawMessage, nil
}

func normalizeAtlasCloudToolCalls(toolCalls []atlasToolCall) []map[string]any {
	out := make([]map[string]any, 0, len(toolCalls))
	for _, tc := range toolCalls {
		toolType := tc.Type
		if toolType == "" {
			toolType = "function"
		}
		out = append(out, map[string]any{
			"id":   tc.ID,
			"type": toolType,
			"function": map[string]any{
				"name":      tc.Function.Name,
				"arguments": tc.Function.Arguments,
			},
		})
	}
	return out
}

func atlascloudRequestSummary(req map[string]any) string {
	parts := []string{}
	if model, ok := req["model"].(string); ok && model != "" {
		parts = append(parts, "model="+model)
	}
	if messages, ok := req["messages"].([]map[string]any); ok {
		parts = append(parts, fmt.Sprintf("messages=%d", len(messages)))
		if len(messages) > 0 {
			last := messages[len(messages)-1]
			if role, ok := last["role"].(string); ok && role != "" {
				parts = append(parts, "last_role="+role)
			}
			if _, ok := last["tool_call_id"].(string); ok {
				parts = append(parts, "last_has_tool_call_id=true")
			}
		}
	}
	if tools, ok := req["tools"].([]map[string]any); ok {
		names := make([]string, 0, len(tools))
		for _, tool := range tools {
			fn, _ := tool["function"].(map[string]any)
			name, _ := fn["name"].(string)
			if name != "" {
				names = append(names, name)
			}
		}
		parts = append(parts, fmt.Sprintf("tools=%d", len(tools)))
		if len(names) > 0 {
			parts = append(parts, "tool_names="+strings.Join(names, ","))
		}
	}
	if len(parts) == 0 {
		return "request_context=unavailable"
	}
	return strings.Join(parts, " ")
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
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
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
	if httpResp.StatusCode != http.StatusOK {
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
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

const defaultVideoModel = "google/gemini-omni-flash/image-to-video-developer"

// GenerateVideo creates a video using Atlas Cloud's async video API.
// Supports text-to-video and image-to-video depending on whether
// Images are provided in the request.
func (p *Provider) GenerateVideo(ctx context.Context, req *ai.VideoRequest, opts ...ai.GenerateOption) (*ai.VideoResponse, error) {
	model := req.Model
	if model == "" {
		model = defaultVideoModel
	}
	duration := req.Duration
	if duration <= 0 {
		duration = 6
	}
	aspect := req.AspectRatio
	if aspect == "" {
		aspect = "16:9"
	}
	resolution := req.Resolution
	if resolution == "" {
		resolution = "720p"
	}

	apiReq := map[string]any{
		"model":        model,
		"prompt":       req.Prompt,
		"duration":     duration,
		"aspect_ratio": aspect,
		"resolution":   resolution,
		"seed":         -1,
	}
	if len(req.Images) > 0 {
		apiReq["images"] = req.Images
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/api/v1/model/generateVideo"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
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
	if httpResp.StatusCode != http.StatusOK {
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

	pollURL := strings.TrimRight(p.opts.BaseURL, "/") + "/api/v1/model/prediction/" + submitResp.Data.ID

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			result, err := p.pollVideo(ctx, pollURL)
			if err != nil {
				return nil, err
			}
			if result != nil {
				return result, nil
			}
		}
	}
}

func (p *Provider) pollVideo(ctx context.Context, url string) (*ai.VideoResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
	case "completed", "succeeded":
		if len(pollResp.Data.Outputs) == 0 {
			return nil, fmt.Errorf("video completed but no outputs returned")
		}
		return &ai.VideoResponse{URL: pollResp.Data.Outputs[0]}, nil
	case "failed":
		return nil, fmt.Errorf("video generation failed: %s", pollResp.Data.Error)
	default:
		return nil, nil
	}
}
