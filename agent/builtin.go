package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"go-micro.dev/v5/ai"
	codecBytes "go-micro.dev/v5/codec/bytes"
	"go-micro.dev/v5/store"
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
					"description": "Optional. The agent or service name best suited to the subtask.",
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
			r, c := a.handlePlan(input)
			return r, c, true
		case toolDelegate:
			r, c := a.handleDelegate(input)
			return r, c, true
		}
		return nil, "", false
	}
	return builtinTools(), handle
}

// toolHandler returns the agent's tool-call handler. It intercepts the
// built-in tools and falls through to RPC service execution for the
// rest. Ephemeral sub-agents get the bare service handler so they can
// neither plan nor re-delegate (which prevents runaway recursion).
func (a *agentImpl) toolHandler() ai.ToolHandler {
	base := a.tools.Handler()
	if a.ephemeral {
		return base
	}
	return func(name string, input map[string]any) (any, string) {
		// plan is internal bookkeeping, not an action — never gated.
		if name == toolPlan {
			return a.handlePlan(input)
		}

		// Stopping condition: bound the number of actions per Ask.
		if a.opts.MaxSteps > 0 {
			a.steps++
			if a.steps > a.opts.MaxSteps {
				return errResult(fmt.Sprintf(
					"step limit reached (%d). Do not call any more tools; stop and summarize what you have so far.",
					a.opts.MaxSteps))
			}
		}

		// Human-in-the-loop / policy: gate the action before it runs.
		if a.opts.Approve != nil {
			if ok, reason := a.opts.Approve(name, input); !ok {
				msg := "tool call was not approved"
				if reason != "" {
					msg += ": " + reason
				}
				return errResult(msg)
			}
		}

		if name == toolDelegate {
			return a.handleDelegate(input)
		}
		return base(name, input)
	}
}

// handlePlan persists the supplied plan to the agent's memory and
// echoes it back so the model can see the stored state.
func (a *agentImpl) handlePlan(input map[string]any) (any, string) {
	data, err := json.Marshal(input)
	if err != nil {
		return errResult("invalid plan: " + err.Error())
	}
	if a.opts.Store != nil {
		a.opts.Store.Write(&store.Record{Key: a.planKey(), Value: data})
	}
	return input, string(data)
}

// handleDelegate hands a subtask to another agent. Delegate-first:
// if 'to' names a registered agent, it is called via RPC. Otherwise an
// ephemeral sub-agent is created with a fresh, isolated context, asked
// the subtask, and its reply returned.
func (a *agentImpl) handleDelegate(input map[string]any) (any, string) {
	task, _ := input["task"].(string)
	if task == "" {
		return errResult("task is required")
	}
	to, _ := input["to"].(string)

	// Delegate-first: an existing agent that owns the domain handles it.
	if to != "" && a.isAgent(to) {
		reply, err := a.callAgentRPC(context.Background(), to, task)
		if err != nil {
			return errResult("delegate to agent " + to + ": " + err.Error())
		}
		out := map[string]any{"agent": to, "reply": reply}
		b, _ := json.Marshal(out)
		return out, string(b)
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

	resp, err := sub.Ask(context.Background(), task)
	if err != nil {
		return errResult("sub-agent: " + err.Error())
	}
	out := map[string]any{"reply": resp.Reply}
	b, _ := json.Marshal(out)
	return out, string(b)
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

func (a *agentImpl) planKey() string {
	return "agent/" + a.opts.Name + "/plan"
}

// loadPlan returns the stored plan as a JSON string, or "" if none.
func (a *agentImpl) loadPlan() string {
	if a.opts.Store == nil {
		return ""
	}
	recs, err := a.opts.Store.Read(a.planKey())
	if err != nil || len(recs) == 0 {
		return ""
	}
	return string(recs[0].Value)
}

func errResult(msg string) (any, string) {
	m := map[string]string{"error": msg}
	b, _ := json.Marshal(m)
	return m, string(b)
}
