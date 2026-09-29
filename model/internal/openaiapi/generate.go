package openaiapi

import (
	"context"
	"fmt"

	"go-micro.dev/v6/model"
)

// MaxToolRounds bounds the legacy v6 tool loop shared by compatible providers.
// Agent guardrails can impose additional limits on individual tool executions.
const MaxToolRounds = 12

// Generate preserves history, options, and tool definitions across model turns.
// call performs one provider request; automatic tool execution remains supported
// here for callers of the v6 model API that supply a ToolHandler.
func Generate(ctx context.Context, opts model.Options, req *model.Request, call func(context.Context, map[string]any) (*model.Response, map[string]any, error)) (*model.Response, error) {
	messages := Messages(req)
	resp, raw, err := call(ctx, Request(opts, messages, req.Tools))
	if err != nil {
		return nil, err
	}
	if opts.ToolHandler == nil {
		return resp, nil
	}

	pending := resp.ToolCalls
	for round := 0; len(pending) > 0 && round < MaxToolRounds; round++ {
		messages = append(messages, map[string]any{
			"role":       "assistant",
			"content":    raw["content"],
			"tool_calls": raw["tool_calls"],
		})
		for _, tc := range pending {
			content := opts.ToolHandler(ctx, tc).Content
			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": tc.ID,
				"content":      content,
			})
		}

		next, nextRaw, err := call(ctx, Request(opts, messages, req.Tools))
		if err != nil {
			return nil, fmt.Errorf("tool follow-up: %w", err)
		}
		if next.Reply != "" {
			resp.Answer = next.Reply
		}
		resp.StopReason = next.StopReason
		pending, raw = next.ToolCalls, nextRaw
		resp.ToolCalls = append(resp.ToolCalls, next.ToolCalls...)
	}
	return resp, nil
}
