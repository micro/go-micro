package openaiapi

import "go-micro.dev/v6/model"

// Messages builds the shared chat history for Generate and Stream requests.
func Messages(req *model.Request) []map[string]any {
	messages := []map[string]any{{"role": "system", "content": req.SystemPrompt}}
	for _, message := range req.Messages {
		messages = append(messages, map[string]any{"role": message.Role, "content": message.Content})
	}
	if req.Prompt != "" {
		messages = append(messages, map[string]any{"role": "user", "content": req.Prompt})
	}
	return messages
}

// Request preserves provider options on initial and follow-up chat requests.
// Pass nil tools for text-only streams, which do not execute tool calls.
func Request(opts model.Options, messages []map[string]any, tools []model.Tool) map[string]any {
	request := map[string]any{"model": opts.Model, "messages": messages}
	if opts.MaxTokens > 0 {
		request["max_tokens"] = opts.MaxTokens
	}
	if opts.Effort != "" {
		request["reasoning_effort"] = opts.Effort
	}
	if len(tools) > 0 {
		definitions := make([]map[string]any, 0, len(tools))
		for _, tool := range tools {
			definitions = append(definitions, map[string]any{"type": "function", "function": map[string]any{"name": tool.Name, "description": tool.Description, "parameters": map[string]any{"type": "object", "properties": tool.Properties}}})
		}
		request["tools"] = definitions
	}
	return request
}
