// Package openaiapi implements the shared OpenAI-compatible chat client
// used by the ai provider shells (openai, groq, mistral, together,
// minimax). It owns transport, request building, SSE parsing, response
// parsing, and the tool-result loop exactly once.
package openaiapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-micro.dev/v6/ai"
)

// Config describes one OpenAI-compatible chat provider. Every behavioral
// divergence between the provider shells collapses into a flag here.
type Config struct {
	// Name is the provider name; backs Model.String() ("groq", "openai", ...).
	Name string
	// DefaultBase is the default BaseURL when ai.WithBaseURL is not set.
	DefaultBase string
	// DefaultModel is the default model when ai.WithModel is not set.
	DefaultModel string
	// Path is the chat endpoint relative to BaseURL. Defaults to "/v1/chat/completions".
	Path string
	// UseMessages makes Generate honor Request.Messages (multi-turn),
	// max_tokens, and reasoning_effort. Off preserves the old single-turn
	// system+prompt behavior of groq/mistral/together.
	UseMessages bool
	// ParseUsage maps provider token usage into Response.Usage.
	ParseUsage bool
}

// Client satisfies ai.Model and ai.ImageModel.
// Use New to construct.
type Client struct {
	cfg  Config
	opts ai.Options
}

// New is the only entry point for an OpenAI-compatible chat provider.
func New(cfg Config, opts ...ai.Option) *Client {
	o := ai.NewOptions(opts...)
	if o.Model == "" {
		o.Model = cfg.DefaultModel
	}
	if o.BaseURL == "" {
		o.BaseURL = cfg.DefaultBase
	}
	if cfg.Path == "" {
		cfg.Path = "/v1/chat/completions"
	}
	return &Client{cfg: cfg, opts: o}
}

func (c *Client) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&c.opts)
	}
	return nil
}

func (c *Client) Options() ai.Options { return c.opts }
func (c *Client) String() string      { return c.cfg.Name }

// Generate performs a chat completion, executing tools when a ToolHandler is set.
func (c *Client) Generate(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (*ai.Response, error) {
	apiReq := c.chatRequest(req)
	resp, rawMessage, err := c.callAPI(ctx, apiReq)
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
	if c.opts.ToolHandler != nil {
		messages := apiReq["messages"].([]map[string]any)
		// Copied rather than aliased: append on a slice that shares an array
		// with messages would overwrite it on a later round.
		followUpMessages := append([]map[string]any(nil), messages...)
		pending := resp.ToolCalls
		raw := rawMessage
		for round := 0; len(pending) > 0 && round < MaxToolRounds; round++ {
			followUpMessages = append(followUpMessages, map[string]any{
				"role":       "assistant",
				"content":    raw["content"],
				"tool_calls": raw["tool_calls"],
			})
			for i, tc := range pending {
				tr := c.opts.ToolHandler(ctx, tc)
				pending[i].Result = tr.Content
				followUpMessages = append(followUpMessages, map[string]any{
					"role":         "tool",
					"tool_call_id": tc.ID,
					"content":      tr.Content,
				})
			}

			followUpReq := c.followUpRequest(followUpMessages)
			if len(req.Tools) > 0 {
				followUpReq["tools"] = buildTools(req.Tools)
			}

			followUpResp, followUpRaw, err := c.callAPI(ctx, followUpReq)
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

// MaxToolRounds bounds the tool-execution loop in a single Generate. Each
// round is a model call plus the tools it asks for, so this is the ceiling on
// one question's cost as well as its length; it is high enough that no honest
// piece of multi-step work reaches it.
const MaxToolRounds = 12

// Stream opens an SSE stream over the chat endpoint.
func (c *Client) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
	messages := []map[string]any{{"role": "system", "content": req.SystemPrompt}}
	for _, m := range req.Messages {
		messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
	}
	if req.Prompt != "" {
		messages = append(messages, map[string]any{"role": "user", "content": req.Prompt})
	}
	apiReq := map[string]any{
		"model":          c.opts.Model,
		"messages":       messages,
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	if c.opts.MaxTokens > 0 {
		apiReq["max_tokens"] = c.opts.MaxTokens
	}
	if c.opts.Effort != "" {
		apiReq["reasoning_effort"] = c.opts.Effort
	}
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stream request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, c.cfg.Path, reqBody)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	httpResp, err := c.roundTripper().RoundTrip(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream API request failed: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, ai.NewHTTPError(httpResp, respBody)
	}
	return &StreamReader{body: httpResp.Body, scanner: bufio.NewScanner(httpResp.Body)}, nil
}

// GenerateImage generates an image via /v1/images/generations.
func (c *Client) GenerateImage(ctx context.Context, req *ai.ImageRequest, opts ...ai.GenerateOption) (*ai.ImageResponse, error) {
	model := req.Model
	if model == "" {
		model = "gpt-image-1"
	}
	n := req.N
	if n <= 0 {
		n = 1
	}
	apiReq := map[string]any{"model": model, "prompt": req.Prompt, "n": n}
	if req.Size != "" {
		apiReq["size"] = req.Size
	}
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/v1/images/generations", reqBody)
	if err != nil {
		return nil, err
	}

	httpResp, err := c.roundTripper().RoundTrip(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return nil, ai.NewHTTPError(httpResp, respBody)
	}

	var imgResp struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &imgResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	response := &ai.ImageResponse{}
	for _, d := range imgResp.Data {
		response.Images = append(response.Images, ai.Image{URL: d.URL, Base64: d.B64JSON})
	}
	return response, nil
}

// chatRequest builds the non-streaming chat body, honoring the UseMessages flag.
func (c *Client) chatRequest(req *ai.Request) map[string]any {
	var messages []map[string]any
	if c.cfg.UseMessages {
		messages = append(messages, map[string]any{"role": "system", "content": req.SystemPrompt})
		for _, m := range req.Messages {
			messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
		}
		if req.Prompt != "" {
			messages = append(messages, map[string]any{"role": "user", "content": req.Prompt})
		}
	} else {
		messages = []map[string]any{
			{"role": "system", "content": req.SystemPrompt},
			{"role": "user", "content": req.Prompt},
		}
	}
	apiReq := map[string]any{"model": c.opts.Model, "messages": messages}
	if c.cfg.UseMessages {
		if c.opts.MaxTokens > 0 {
			apiReq["max_tokens"] = c.opts.MaxTokens
		}
		if c.opts.Effort != "" {
			apiReq["reasoning_effort"] = c.opts.Effort
		}
	}
	if len(req.Tools) > 0 {
		apiReq["tools"] = buildTools(req.Tools)
	}
	return apiReq
}

func (c *Client) followUpRequest(messages []map[string]any) map[string]any {
	apiReq := map[string]any{"model": c.opts.Model, "messages": messages}
	if c.cfg.UseMessages {
		if c.opts.MaxTokens > 0 {
			apiReq["max_tokens"] = c.opts.MaxTokens
		}
		if c.opts.Effort != "" {
			apiReq["reasoning_effort"] = c.opts.Effort
		}
	}
	return apiReq
}

func (c *Client) callAPI(ctx context.Context, apiReq map[string]any) (*ai.Response, map[string]any, error) {
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, c.cfg.Path, reqBody)
	if err != nil {
		return nil, nil, err
	}

	httpResp, err := c.roundTripper().RoundTrip(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return nil, nil, ai.NewHTTPError(httpResp, respBody)
	}

	var chatResp struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
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
	if c.cfg.ParseUsage {
		response.Usage = ai.Usage{
			InputTokens:  chatResp.Usage.PromptTokens,
			OutputTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:  chatResp.Usage.TotalTokens,
		}
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

func (c *Client) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	apiURL := strings.TrimRight(c.opts.BaseURL, "/") + path
	httpReq, err := http.NewRequestWithContext(ctx, method, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.opts.APIKey)
	return httpReq, nil
}

// ponytail: use DefaultTransport directly; identical to DefaultClient's
// transport minus redirect-following, which chat APIs never need.
func (c *Client) roundTripper() http.RoundTripper {
	if c.opts.Transport != nil {
		return c.opts.Transport
	}
	return http.DefaultTransport
}

func buildTools(tools []ai.Tool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
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
	return out
}
