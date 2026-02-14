// Package anthropic implements the Anthropic Claude model provider
package anthropic

import (
	"encoding/json"
	"strings"

	"go-micro.dev/v5/model"
)

func init() {
	model.Register("anthropic", func(opts model.Options) model.Model {
		return NewProvider(opts)
	})
}

// Provider implements the model.Model interface for Anthropic Claude
type Provider struct {
	options model.Options
}

// NewProvider creates a new Anthropic provider
func NewProvider(opts model.Options) *Provider {
	return &Provider{
		options: opts,
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "anthropic"
}

// DefaultModel returns the default model for Anthropic
func (p *Provider) DefaultModel() string {
	return "claude-sonnet-4-20250514"
}

// DefaultBaseURL returns the default API base URL for Anthropic
func (p *Provider) DefaultBaseURL() string {
	return "https://api.anthropic.com"
}

// BuildRequest constructs a request payload for Anthropic's Messages API
func (p *Provider) BuildRequest(prompt string, systemPrompt string, tools []model.Tool, messages []model.Message) ([]byte, error) {
	// Build tools for Anthropic format
	var anthropicTools []map[string]any
	for _, t := range tools {
		anthropicTools = append(anthropicTools, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"input_schema": map[string]any{
				"type":       "object",
				"properties": t.Properties,
			},
		})
	}

	// Build request
	req := map[string]any{
		"model":      p.DefaultModel(), // Will be overridden by caller if needed
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	}

	if len(anthropicTools) > 0 {
		req["tools"] = anthropicTools
	}

	return json.Marshal(req)
}

// ParseResponse parses the Anthropic API response
func (p *Provider) ParseResponse(body []byte) (*model.Response, error) {
	var anthropicResp struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}

	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, err
	}

	response := &model.Response{
		RawContent: anthropicResp.Content,
	}

	// Extract text reply
	var replyParts []string
	for _, block := range anthropicResp.Content {
		if block.Type == "text" && block.Text != "" {
			replyParts = append(replyParts, block.Text)
		}
	}
	if len(replyParts) > 0 {
		response.Reply = strings.Join(replyParts, "\n")
	}

	// Extract tool calls
	for _, block := range anthropicResp.Content {
		if block.Type == "tool_use" {
			var input map[string]any
			if err := json.Unmarshal(block.Input, &input); err != nil {
				input = map[string]any{}
			}
			response.ToolCalls = append(response.ToolCalls, model.ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: input,
			})
		}
	}

	return response, nil
}

// BuildFollowUpRequest constructs a follow-up request with tool results
func (p *Provider) BuildFollowUpRequest(prompt string, systemPrompt string, originalResponse *model.Response, toolResults []model.ToolResult) ([]byte, error) {
	// Build tool result blocks
	var toolResultBlocks []map[string]any
	for _, tr := range toolResults {
		toolResultBlocks = append(toolResultBlocks, map[string]any{
			"type":        "tool_result",
			"tool_use_id": tr.ID,
			"content":     tr.Content,
		})
	}

	// Build follow-up request
	req := map[string]any{
		"model":      p.DefaultModel(), // Will be overridden by caller if needed
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
			{"role": "assistant", "content": originalResponse.RawContent},
			{"role": "user", "content": toolResultBlocks},
		},
	}

	return json.Marshal(req)
}

// ParseFollowUpResponse parses the follow-up response
func (p *Provider) ParseFollowUpResponse(body []byte) (string, error) {
	var followUpResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &followUpResp); err != nil {
		return "", err
	}

	var answerParts []string
	for _, block := range followUpResp.Content {
		if block.Type == "text" && block.Text != "" {
			answerParts = append(answerParts, block.Text)
		}
	}

	return strings.Join(answerParts, "\n"), nil
}

// SetAuthHeaders sets the required authentication headers for Anthropic
func (p *Provider) SetAuthHeaders(headers map[string]string, apiKey string) {
	headers["x-api-key"] = apiKey
	headers["anthropic-version"] = "2023-06-01"
}

// GetAPIEndpoint returns the full API endpoint URL for Anthropic
func (p *Provider) GetAPIEndpoint(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/v1/messages"
}
