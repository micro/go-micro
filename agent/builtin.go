package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-micro.dev/v6/ai"
	codecBytes "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/gateway/a2a"
	"go-micro.dev/v6/store"
)

// Built-in agent tools. These are not service endpoints — they are
// capabilities the agent has over itself: maintaining a plan in its
// memory, and delegating a subtask to another agent.
//
// They are plain tools, wired into the agent's tool handler alongside
// the discovered service tools. There is no separate harness or graph:
// the LLM calls them like any other tool.
const (
	toolPlan     = "plan"
	toolDelegate = "delegate"
)

// builtinTools returns the tool definitions exposed to the model in
// addition to the agent's scoped service tools.
func builtinTools() []ai.Tool {
	return []ai.Tool{
		{
			Name:         toolPlan,
			OriginalName: toolPlan,
			Description: "Record or update your plan as an ordered list of steps before doing multi-step work. " +
				"Call this whenever the plan changes. The plan is saved to your memory and shown back to you on later turns.",
			Properties: map[string]any{
				"steps": map[string]any{
					"type": "array",
					"description": "Ordered plan steps. Each step has a 'task' (string) and a " +
						"'status' (one of: pending, in_progress, done).",
				},
			},
		},
		{
			Name:         toolDelegate,
			OriginalName: toolDelegate,
			Description: "Delegate a self-contained subtask to another agent. If 'to' names an agent that already " +
				"manages the relevant services, that agent handles it; otherwise a focused sub-agent is created for the " +
				"subtask. The sub-agent works in an isolated context and returns only its result. Use this to keep your " +
				"own context focused and to let domain experts handle their own services.",
			Properties: map[string]any{
				"task": map[string]any{
					"type":        "string",
					"description": "The subtask to delegate, described completely and self-contained.",
				},
				"to": map[string]any{
					"type":        "string",
					"description": "Optional. The agent or service name best suited to the subtask, or the URL of an external agent that speaks the A2A protocol.",
				},
			},
		},
	}
}

// Builtins returns the built-in agent tools (plan, delegate) together
// with a handler for them, so the same capabilities can be wired into a
// tool loop that isn't a running Agent — for example the `micro chat`
// fallback. The handler's third return value is false when the name is
// not a built-in, so callers can fall through to their own tools.
//
// Configure it with the same options as an Agent (Name, Provider,
// WithStore, WithRegistry, WithClient, ...); these back plan's memory
// and delegate's RPC/sub-agent behaviour.
func Builtins(opts ...Option) (tools []ai.Tool, handle func(name string, input map[string]any) (result any, content string, ok bool)) {
	a := &agentImpl{opts: newOptions(opts...)}
	handle = func(name string, input map[string]any) (any, string, bool) {
		switch name {
		case toolPlan:
			r := a.handlePlan(ai.ToolCall{Name: name, Input: input})
			return r.Value, r.Content, true
		case toolDelegate:
			r := a.handleDelegate(ai.ToolCall{Name: name, Input: input})
			return r.Value, r.Content, true
		}
		return nil, "", false
	}
	return builtinTools(), handle
}

// toolHandler returns the agent's tool-call handler, composed as a stack
// of wrappers around a base handler — the same middleware shape as
// client/server wrappers. The base executes the call (custom tools,
// delegate, or RPC); the built-in guardrails wrap it; developer wrappers
// (WrapTool) wrap those, outermost, so they observe every call and its
// result including guardrail refusals. Ephemeral sub-agents get the bare
// service handler so they can neither plan nor re-delegate (which
// prevents runaway recursion).
func (a *agentImpl) toolHandler() ai.ToolHandler {
	if a.ephemeral {
		return a.tools.Handler()
	}

	// Innermost first: base, then guardrails (approve → loop → step →
	// plan), then developer wrappers outermost. Wrapping reverses order,
	// so the result runs plan → step → loop → approve → base.
	h := a.baseHandler()
	h = a.approveWrap(h)
	h = a.loopWrap(h)
	h = a.stepWrap(h)
	h = a.planWrap(h)
	for i := len(a.opts.wrappers) - 1; i >= 0; i-- {
		h = a.opts.wrappers[i](h)
	}
	return h
}

// baseHandler executes a tool call: a developer custom tool, the built-in
// delegate, or an RPC to the service. It is the innermost handler.
func (a *agentImpl) baseHandler() ai.ToolHandler {
	rpc := a.tools.Handler()
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		for i := range a.opts.tools {
			if a.opts.tools[i].def.Name == call.Name {
				out, err := a.opts.tools[i].handler(ctx, call.Input)
				if err != nil {
					return errResult(call.ID, err.Error())
				}
				return ai.ToolResult{ID: call.ID, Value: out, Content: out}
			}
		}
		if call.Name == toolDelegate {
			return a.handleDelegate(call)
		}
		return rpc(ctx, call)
	}
}

// planWrap handles the plan tool inline. plan is internal bookkeeping,
// not an action — it is never counted, loop-checked, or gated.
func (a *agentImpl) planWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if call.Name == toolPlan {
			return a.handlePlan(call)
		}
		return next(ctx, call)
	}
}

// stepWrap bounds the number of actions per Ask (MaxSteps).
func (a *agentImpl) stepWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if a.opts.MaxSteps > 0 {
			a.steps++
			if a.steps > a.opts.MaxSteps {
				return refused(call.ID, ai.RefusedMaxSteps, fmt.Sprintf(
					"step limit reached (%d). Do not call any more tools; stop and summarize what you have so far.",
					a.opts.MaxSteps))
			}
		}
		return next(ctx, call)
	}
}

// loopWrap stops the agent repeating an identical action that makes no
// progress (which the step count alone won't catch).
func (a *agentImpl) loopWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if a.opts.LoopLimit > 0 {
			if a.calls == nil {
				a.calls = map[string]int{}
			}
			args, _ := json.Marshal(call.Input)
			fp := call.Name + ":" + string(args)
			a.calls[fp]++
			if a.calls[fp] > a.opts.LoopLimit {
				return refused(call.ID, ai.RefusedLoop, fmt.Sprintf(
					"loop detected: you have already called %q with the same arguments %d times and the result will not change. Stop repeating it — try a different approach, or finish with what you have.",
					call.Name, a.opts.LoopLimit))
			}
		}
		return next(ctx, call)
	}
}

// approveWrap gates each action before it runs (ApproveTool).
func (a *agentImpl) approveWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if a.opts.Approve != nil {
			if ok, reason := a.opts.Approve(call.Name, call.Input); !ok {
				msg := "tool call was not approved"
				if reason != "" {
					msg += ": " + reason
				}
				return refused(call.ID, ai.RefusedApproval, msg)
			}
		}
		return next(ctx, call)
	}
}

// handlePlan persists the supplied plan to the agent's memory and
// echoes it back so the model can see the stored state.
func (a *agentImpl) handlePlan(call ai.ToolCall) ai.ToolResult {
	data, err := json.Marshal(call.Input)
	if err != nil {
		return errResult(call.ID, "invalid plan: "+err.Error())
	}
	a.stateStore().Write(&store.Record{Key: planKey, Value: data})
	return ai.ToolResult{ID: call.ID, Value: call.Input, Content: string(data)}
}

// handleDelegate hands a subtask to another agent. Delegate-first:
// if 'to' names a registered agent, it is called via RPC. Otherwise an
// ephemeral sub-agent is created with a fresh, isolated context, asked
// the subtask, and its reply returned.
func (a *agentImpl) handleDelegate(call ai.ToolCall) ai.ToolResult {
	input := call.Input
	task, _ := input["task"].(string)
	if task == "" {
		return errResult(call.ID, "task is required")
	}
	to, _ := input["to"].(string)

	// An external agent on another framework, addressed by A2A URL.
	if strings.HasPrefix(to, "http://") || strings.HasPrefix(to, "https://") {
		reply, err := a2a.NewClient(to).Send(context.Background(), task)
		if err != nil {
			return errResult(call.ID, "delegate to A2A agent "+to+": "+err.Error())
		}
		out := map[string]any{"agent": to, "reply": reply}
		b, _ := json.Marshal(out)
		return ai.ToolResult{ID: call.ID, Value: out, Content: string(b)}
	}

	// Delegate-first: an existing agent that owns the domain handles it.
	if to != "" && a.isAgent(to) {
		reply, err := a.callAgentRPC(context.Background(), to, task)
		if err != nil {
			return errResult(call.ID, "delegate to agent "+to+": "+err.Error())
		}
		out := map[string]any{"agent": to, "reply": reply}
		b, _ := json.Marshal(out)
		return ai.ToolResult{ID: call.ID, Value: out, Content: string(b)}
	}

	// Otherwise create a focused, ephemeral sub-agent. Fresh context:
	// it loads no history and persists none.
	var svcs []string
	if to != "" {
		svcs = []string{to}
	}
	sub := newEphemeral(
		Name(a.opts.Name+".sub"),
		Services(svcs...),
		Prompt("You are a sub-agent handling a single delegated subtask. "+
			"Complete it using the available tools and report the result concisely."),
		Provider(a.opts.Provider),
		Model(a.opts.Model),
		APIKey(a.opts.APIKey),
		WithRegistry(a.opts.Registry),
		WithClient(a.opts.Client),
		WithStore(a.opts.Store),
	)
	// Record lineage so the sub-agent's tool calls carry this run as parent.
	sub.parentRunID = a.runID

	resp, err := sub.Ask(context.Background(), task)
	if err != nil {
		return errResult(call.ID, "sub-agent: "+err.Error())
	}
	out := map[string]any{"reply": resp.Reply}
	b, _ := json.Marshal(out)
	return ai.ToolResult{ID: call.ID, Value: out, Content: string(b)}
}

// isAgent reports whether name resolves to a registered agent (a
// service advertising type=agent in its metadata).
func (a *agentImpl) isAgent(name string) bool {
	if a.opts.Registry == nil {
		return false
	}
	recs, err := a.opts.Registry.GetService(name)
	if err != nil || len(recs) == 0 {
		return false
	}
	if recs[0].Metadata != nil && recs[0].Metadata["type"] == "agent" {
		return true
	}
	for _, n := range recs[0].Nodes {
		if n.Metadata != nil && n.Metadata["type"] == "agent" {
			return true
		}
	}
	return false
}

// callAgentRPC calls another agent's Agent.Chat endpoint and returns
// its reply.
func (a *agentImpl) callAgentRPC(ctx context.Context, name, msg string) (string, error) {
	body, _ := json.Marshal(map[string]string{"message": msg})
	req := a.opts.Client.NewRequest(name, "Agent.Chat", &codecBytes.Frame{Data: body})
	var rsp codecBytes.Frame
	if err := a.opts.Client.Call(ctx, req, &rsp); err != nil {
		return "", err
	}
	var out struct {
		Reply string `json:"reply"`
	}
	if err := json.Unmarshal(rsp.Data, &out); err != nil {
		return "", err
	}
	return out.Reply, nil
}

// planKey is the record key for an agent's plan within its scoped store.
const planKey = "plan"

// loadPlan returns the stored plan as a JSON string, or "" if none.
func (a *agentImpl) loadPlan() string {
	recs, err := a.stateStore().Read(planKey)
	if err != nil || len(recs) == 0 {
		return ""
	}
	return string(recs[0].Value)
}

func errResult(id, msg string) ai.ToolResult {
	m := map[string]string{"error": msg}
	b, _ := json.Marshal(m)
	return ai.ToolResult{ID: id, Value: m, Content: string(b)}
}

// refused is an error result a guardrail returns, tagged with a structured
// reason (ai.Refused*) so a tool wrapper can react to it without parsing
// the message.
func refused(id, reason, msg string) ai.ToolResult {
	r := errResult(id, msg)
	r.Refused = reason
	return r
}
