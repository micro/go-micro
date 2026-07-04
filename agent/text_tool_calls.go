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
var taggedToolCallBlock = regexp.MustCompile(`(?s)<[^<>]*(?:tool_call|tool_calls|function=)[^<>]*>(.*?)</[^<>]*>`)
var singleTaggedToolCall = regexp.MustCompile(`(?s)<(tool_call\b[^<>]*|[^<>]*function=[^<>]*)>(.*?)</[^<>]*>`)

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
	allowed := textToolNames(tools)
	if len(allowed) == 0 {
		return nil
	}

	if calls := decodeTaggedTextToolCalls(text, allowed); len(calls) > 0 {
		return calls
	}
	for _, candidate := range jsonCandidates(text) {
		if calls := decodeTextToolCalls(candidate, allowed); len(calls) > 0 {
			return calls
		}
	}
	return nil
}

func textToolNames(tools []ai.Tool) map[string]string {
	allowed := map[string]string{}
	for _, tool := range tools {
		addTextToolName(allowed, tool.Name, tool.Name)
		if tool.OriginalName != "" {
			addTextToolName(allowed, tool.OriginalName, tool.Name)
		}
	}
	return allowed
}

func addTextToolName(allowed map[string]string, name, canonical string) {
	if name == "" || canonical == "" {
		return
	}
	allowed[name] = canonical
	// Some OpenAI-compatible models describe an idempotent Add endpoint as a
	// creation action and emit the otherwise-correct service tool with a Create
	// suffix in text-only tool-call markup. Keep the fallback bounded by the
	// offered service tool prefix so ordinary unknown tools remain ignored.
	for _, suffix := range []string{"_Add", ".Add"} {
		if strings.HasSuffix(name, suffix) {
			allowed[strings.TrimSuffix(name, suffix)+strings.Replace(suffix, "Add", "Create", 1)] = canonical
		}
	}
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
	for _, match := range taggedToolCallBlock.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			out = append(out, strings.TrimSpace(match[1]))
		}
	}
	if start, end := strings.IndexAny(text, "[{"), strings.LastIndexAny(text, "]}"); start >= 0 && end > start {
		out = append(out, strings.TrimSpace(text[start:end+1]))
	}
	return out
}

func decodeTextToolCalls(candidate string, allowed map[string]string) []ai.ToolCall {
	var root any
	if err := json.Unmarshal([]byte(candidate), &root); err != nil {
		return nil
	}
	return collectTextToolCalls(root, allowed)
}

func collectTextToolCalls(v any, allowed map[string]string) []ai.ToolCall {
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
		if name == "" || allowed[name] == "" || input == nil {
			return nil
		}
		id := call.ID
		if id == "" {
			id = fmt.Sprintf("text-call-%s", strings.ReplaceAll(name, ".", "_"))
		}
		return []ai.ToolCall{{ID: id, Name: allowed[name], Input: input}}
	default:
		return nil
	}
}

func decodeTaggedTextToolCalls(text string, allowed map[string]string) []ai.ToolCall {
	var out []ai.ToolCall
	for _, match := range singleTaggedToolCall.FindAllStringSubmatch(text, -1) {
		if len(match) < 3 {
			continue
		}
		tag, body := match[1], strings.TrimSpace(match[2])
		if calls := decodeTextToolCalls(body, allowed); len(calls) > 0 {
			out = append(out, calls...)
			continue
		}
		if calls := decodeTaggedTextToolCalls(body, allowed); len(calls) > 0 {
			out = append(out, calls...)
			continue
		}
		name := taggedToolName(tag)
		if name == "" || allowed[name] == "" {
			continue
		}
		var input map[string]any
		if err := json.Unmarshal([]byte(body), &input); err != nil || input == nil {
			continue
		}
		out = append(out, ai.ToolCall{
			ID:    fmt.Sprintf("text-call-%s", strings.ReplaceAll(name, ".", "_")),
			Name:  allowed[name],
			Input: input,
		})
	}
	return out
}

func taggedToolName(tag string) string {
	for _, marker := range []string{"function=", "name=", "tool="} {
		if idx := strings.Index(tag, marker); idx >= 0 {
			name := strings.TrimSpace(tag[idx+len(marker):])
			name = strings.Trim(name, `"'`)
			if end := strings.IndexAny(name, " \t\r\n>"); end >= 0 {
				name = name[:end]
			}
			return strings.Trim(name, `"'`)
		}
	}
	return ""
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
