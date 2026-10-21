package openaiapi

import (
	"strings"

	"go-micro.dev/v6/model"
)

// ChatUsage is the usage object of an OpenAI-compatible chat completion.
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Usage converts the response usage to model.Usage.
func (u ChatUsage) Usage() model.Usage {
	return model.Usage{InputTokens: u.PromptTokens, OutputTokens: u.CompletionTokens, TotalTokens: u.TotalTokens}
}

// CheckOutputLimit returns model.ErrOutputLimit when a response stopped at the
// token limit without any visible content or tool call.
func CheckOutputLimit(finishReason, content string, toolCalls int) error {
	if finishReason == "length" && strings.TrimSpace(content) == "" && toolCalls == 0 {
		return model.ErrOutputLimit
	}
	return nil
}
