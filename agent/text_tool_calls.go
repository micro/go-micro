package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"go-micro.dev/v6/ai"
)

var fencedJSONBlock = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")

type textToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Tool      string         `json:"tool"`
	Input     map[string]any `json:"input"`
	Arguments map[string]any `json:"arguments"`
}

// executeTextToolCalls is a compatibility fallback for providers that return a
// tool call as text JSON instead of a structured tool_calls field. It only runs
// calls whose names match the tools offered to the model, so ordinary JSON
// answers are left untouched.
func (a *agentImpl) executeTextToolCalls(ctx context.Context, reply string, tools []ai.Tool) ([]ai.ToolCall, string, bool) {
	calls := parseTextToolCalls(reply, tools)
	if len(calls) == 0 {
		return nil, "", false
	}

	handler := a.toolHandler()
	results := make([]string, 0, len(calls))
	for i := range calls {
		result := handler(ctx, calls[i])
		calls[i].Result = result.Content
		if result.Refused != "" {
			calls[i].Error = result.Refused
		}
		if result.Content != "" {
			results = append(results, result.Content)
		}
	}
	return calls, strings.Join(results, "\n"), true
}

func parseTextToolCalls(text string, tools []ai.Tool) []ai.ToolCall {
	allowed := map[string]bool{}
	for _, tool := range tools {
		allowed[tool.Name] = true
		if tool.OriginalName != "" {
			allowed[tool.OriginalName] = true
		}
	}
	if len(allowed) == 0 {
		return nil
	}

	for _, candidate := range jsonCandidates(text) {
		if calls := decodeTextToolCalls(candidate, allowed); len(calls) > 0 {
			return calls
		}
	}
	return nil
}

func jsonCandidates(text string) []string {
	trimmed := strings.TrimSpace(text)
	var out []string
	if trimmed != "" {
		out = append(out, trimmed)
	}
	for _, match := range fencedJSONBlock.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			out = append(out, strings.TrimSpace(match[1]))
		}
	}
	if start, end := strings.IndexAny(text, "[{"), strings.LastIndexAny(text, "]}"); start >= 0 && end > start {
		out = append(out, strings.TrimSpace(text[start:end+1]))
	}
	return out
}

func decodeTextToolCalls(candidate string, allowed map[string]bool) []ai.ToolCall {
	var root any
	if err := json.Unmarshal([]byte(candidate), &root); err != nil {
		return nil
	}
	return collectTextToolCalls(root, allowed)
}

func collectTextToolCalls(v any, allowed map[string]bool) []ai.ToolCall {
	switch x := v.(type) {
	case []any:
		var out []ai.ToolCall
		for _, item := range x {
			out = append(out, collectTextToolCalls(item, allowed)...)
		}
		return out
	case map[string]any:
		if nested, ok := firstNestedToolCalls(x); ok {
			return collectTextToolCalls(nested, allowed)
		}
		call := mapToTextToolCall(x)
		name := call.Name
		if name == "" {
			name = call.Tool
		}
		input := call.Input
		if input == nil {
			input = call.Arguments
		}
		if name == "" || !allowed[name] || input == nil {
			return nil
		}
		id := call.ID
		if id == "" {
			id = fmt.Sprintf("text-call-%s", strings.ReplaceAll(name, ".", "_"))
		}
		return []ai.ToolCall{{ID: id, Name: name, Input: input}}
	default:
		return nil
	}
}

func firstNestedToolCalls(m map[string]any) (any, bool) {
	for _, key := range []string{"tool_calls", "toolCalls", "calls"} {
		if v, ok := m[key]; ok {
			return v, true
		}
	}
	return nil, false
}

func mapToTextToolCall(m map[string]any) textToolCall {
	b, _ := json.Marshal(m)
	var call textToolCall
	_ = json.Unmarshal(b, &call)
	return call
}
