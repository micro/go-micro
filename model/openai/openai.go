// Package openai implements the OpenAI model provider
package openai

import (
	"encoding/json"
	"strings"

	"go-micro.dev/v5/model"
)

func init() {
	model.Register("openai", func(opts model.Options) model.Model {
		return NewProvider(opts)
	})
}

// Provider implements the model.Model interface for OpenAI
type Provider struct {
	options model.Options
}

// NewProvider creates a new OpenAI provider
func NewProvider(opts model.Options) *Provider {
	return &Provider{
		options: opts,
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "openai"
}

// DefaultModel returns the default model for OpenAI
func (p *Provider) DefaultModel() string {
	return "gpt-4o"
}

// DefaultBaseURL returns the default API base URL for OpenAI
func (p *Provider) DefaultBaseURL() string {
	return "https://api.openai.com"
}

// BuildRequest constructs a request payload for OpenAI's Chat Completions API
func (p *Provider) BuildRequest(prompt string, systemPrompt string, tools []model.Tool, messages []model.Message) ([]byte, error) {
	// Build tools for OpenAI format
	var openaiTools []map[string]any
	for _, t := range tools {
		openaiTools = append(openaiTools, map[string]any{
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

	// Build messages
	msgs := []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": prompt},
	}

	// Build request
	req := map[string]any{
		"model":    p.DefaultModel(), // Will be overridden by caller if needed
		"messages": msgs,
	}

	if len(openaiTools) > 0 {
		req["tools"] = openaiTools
	}

	return json.Marshal(req)
}

// ParseResponse parses the OpenAI API response
func (p *Provider) ParseResponse(body []byte) (*model.Response, error) {
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

	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Choices) == 0 {
		return &model.Response{}, nil
	}

	choice := chatResp.Choices[0]
	response := &model.Response{
		Reply:      choice.Message.Content,
		RawContent: choice.Message,
	}

	// Extract tool calls
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

	return response, nil
}

// BuildFollowUpRequest constructs a follow-up request with tool results
func (p *Provider) BuildFollowUpRequest(prompt string, systemPrompt string, originalResponse *model.Response, toolResults []model.ToolResult) ([]byte, error) {
	// Build messages
	messages := []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": prompt},
	}

	// Add assistant message with original response
	if rawMsg, ok := originalResponse.RawContent.(struct {
		Content   string `json:"content"`
		ToolCalls []struct {
			ID       string `json:"id"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	}); ok {
		messages = append(messages, map[string]any{
			"role":       "assistant",
			"content":    rawMsg.Content,
			"tool_calls": rawMsg.ToolCalls,
		})
	}

	// Add tool results
	for _, tr := range toolResults {
		messages = append(messages, map[string]any{
			"role":         "tool",
			"tool_call_id": tr.ID,
			"content":      tr.Content,
		})
	}

	// Build request
	req := map[string]any{
		"model":    p.DefaultModel(), // Will be overridden by caller if needed
		"messages": messages,
	}

	return json.Marshal(req)
}

// ParseFollowUpResponse parses the follow-up response
func (p *Provider) ParseFollowUpResponse(body []byte) (string, error) {
	var followUpChat struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &followUpChat); err != nil {
		return "", err
	}

	if len(followUpChat.Choices) > 0 {
		return followUpChat.Choices[0].Message.Content, nil
	}

	return "", nil
}

// SetAuthHeaders sets the required authentication headers for OpenAI
func (p *Provider) SetAuthHeaders(headers map[string]string, apiKey string) {
	headers["Authorization"] = "Bearer " + apiKey
}

// GetAPIEndpoint returns the full API endpoint URL for OpenAI
func (p *Provider) GetAPIEndpoint(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
}
