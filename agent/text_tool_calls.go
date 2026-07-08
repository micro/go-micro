package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"

	"go-micro.dev/v6/ai"
)

var fencedJSONBlock = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
var taggedToolCallBlock = regexp.MustCompile(`(?s)<[^<>]*(?:tool_call|tool_calls|function=)[^<>]*>(.*?)</[^<>]*>`)
var singleTaggedToolCall = regexp.MustCompile(`(?s)<(tool_call\b[^<>]*|[^<>]*function\s*=[^<>]*)>(.*?)</[^<>]*>`)
var taggedToolNameAttr = regexp.MustCompile(`(?i)(?:function|name|tool)\s*=\s*["\']?([^"\'\s>]+)`)

type textToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Tool      string         `json:"tool"`
	Input     map[string]any `json:"input"`
	Arguments any            `json:"arguments"`
	Function  *textToolCall  `json:"function"`
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

// executeAdditionalTextToolCalls runs text-encoded tool calls that accompany a
// structured tool_calls response. Some OpenAI-compatible providers can mix the
// two forms in a single assistant turn: for example, emitting a native
// conformance_echo call while rendering a follow-up guarded delegate call as
// <tool_call name="delegate">...</tool_call> text. Keep this fallback additive
// and de-duplicate calls already represented in the structured tool_calls list.
func (a *agentImpl) executeAdditionalTextToolCalls(ctx context.Context, reply string, tools []ai.Tool, existing []ai.ToolCall) ([]ai.ToolCall, string, bool) {
	calls := parseTextToolCalls(reply, tools)
	if len(calls) == 0 {
		return nil, "", false
	}

	seen := map[string]bool{}
	for _, call := range existing {
		seen[textToolCallKey(call)] = true
	}

	handler := a.toolHandler()
	out := make([]ai.ToolCall, 0, len(calls))
	results := make([]string, 0, len(calls))
	for i := range calls {
		if seen[textToolCallKey(calls[i])] {
			continue
		}
		result := handler(ctx, calls[i])
		calls[i].Result = result.Content
		if result.Refused != "" {
			calls[i].Error = result.Refused
		}
		if result.Content != "" {
			results = append(results, result.Content)
		}
		out = append(out, calls[i])
	}
	return out, strings.Join(results, "\n"), len(out) > 0
}

func textToolCallKey(call ai.ToolCall) string {
	b, _ := json.Marshal(call.Input)
	return call.Name + "\x00" + string(b)
}

func parseTextToolCalls(text string, tools []ai.Tool) []ai.ToolCall {
	text = html.UnescapeString(text)
	allowed := textToolNames(tools)
	if len(allowed) == 0 {
		return nil
	}

	if calls := decodeTaggedTextToolCalls(text, allowed); len(calls) > 0 {
		return calls
	}
	if calls := decodeFunctionTextToolCalls(text, allowed); len(calls) > 0 {
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
		name, input := textToolCallNameAndInput(call)
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

func textToolCallNameAndInput(call textToolCall) (string, map[string]any) {
	name := call.Name
	if name == "" {
		name = call.Tool
	}
	input := call.Input
	if input == nil {
		input = textToolArguments(call.Arguments)
	}
	if call.Function != nil {
		fnName, fnInput := textToolCallNameAndInput(*call.Function)
		if name == "" {
			name = fnName
		}
		if input == nil {
			input = fnInput
		}
	}
	return name, input
}

func textToolArguments(raw any) map[string]any {
	switch args := raw.(type) {
	case map[string]any:
		return args
	case string:
		var input map[string]any
		if err := json.Unmarshal([]byte(args), &input); err == nil {
			return input
		}
	}
	return nil
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
	match := taggedToolNameAttr.FindStringSubmatch(tag)
	if len(match) < 2 {
		return ""
	}
	return strings.Trim(match[1], `"'`)
}

func decodeFunctionTextToolCalls(text string, allowed map[string]string) []ai.ToolCall {
	var out []ai.ToolCall
	for alias, canonical := range allowed {
		for _, body := range functionCallBodies(text, alias) {
			var input map[string]any
			if err := json.Unmarshal([]byte(body), &input); err != nil || input == nil {
				continue
			}
			out = append(out, ai.ToolCall{
				ID:    fmt.Sprintf("text-call-%s", strings.ReplaceAll(alias, ".", "_")),
				Name:  canonical,
				Input: input,
			})
		}
	}
	return out
}

func functionCallBodies(text, name string) []string {
	if name == "" {
		return nil
	}
	var bodies []string
	for searchFrom := 0; searchFrom < len(text); {
		idx := strings.Index(text[searchFrom:], name)
		if idx < 0 {
			break
		}
		start := searchFrom + idx
		open := start + len(name)
		if !isFunctionCallBoundary(text, start, open) {
			searchFrom = start + len(name)
			continue
		}
		bodyStart := open + 1
		bodyEnd, ok := balancedJSONObjectEnd(text, bodyStart)
		if !ok {
			searchFrom = bodyStart
			continue
		}
		bodies = append(bodies, strings.TrimSpace(text[bodyStart:bodyEnd]))
		searchFrom = bodyEnd + 1
	}
	return bodies
}

func isFunctionCallBoundary(text string, start, open int) bool {
	if open >= len(text) || text[open] != '(' {
		return false
	}
	if start > 0 {
		prev := text[start-1]
		if prev == '_' || prev == '.' || prev == '-' || prev == '$' || ('0' <= prev && prev <= '9') || ('A' <= prev && prev <= 'Z') || ('a' <= prev && prev <= 'z') {
			return false
		}
	}
	for i := open + 1; i < len(text); i++ {
		switch text[i] {
		case ' ', '\n', '\r', '\t':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

func balancedJSONObjectEnd(text string, start int) (int, bool) {
	for start < len(text) {
		switch text[start] {
		case ' ', '\n', '\r', '\t':
			start++
		case '{':
			depth := 0
			inString := false
			escaped := false
			for i := start; i < len(text); i++ {
				c := text[i]
				if inString {
					if escaped {
						escaped = false
					} else if c == '\\' {
						escaped = true
					} else if c == '"' {
						inString = false
					}
					continue
				}
				switch c {
				case '"':
					inString = true
				case '{':
					depth++
				case '}':
					depth--
					if depth == 0 {
						for j := i + 1; j < len(text); j++ {
							switch text[j] {
							case ' ', '\n', '\r', '\t':
								continue
							case ')':
								return i + 1, true
							default:
								return 0, false
							}
						}
					}
				}
			}
			return 0, false
		default:
			return 0, false
		}
	}
	return 0, false
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
