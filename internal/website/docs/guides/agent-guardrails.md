---
layout: default
---

# Agent Guardrails

An autonomous agent decides its own actions at runtime, which is what makes it useful — and what makes it risky. The common failure modes are mundane: it loops, repeating the same call without making progress; it runs away, taking far more steps (and cost) than the task warrants; it takes an action that should have had a human or a policy in the way.

Go Micro separates **orchestration** (the model deciding what to do) from **execution safety** (whether a decided action is allowed to run). Every tool call an agent makes passes through one choke point, and that's where the guardrails live — so they apply uniformly to service calls, custom tools, and `delegate`, without touching the model or your services.

## The three agent guardrails

### Stop on count — `MaxSteps`

Bounds the total number of tool executions in a single `Ask`. Once exceeded, further calls are refused and the model is told to stop and summarize. The blunt backstop against runaway cost.

```go
micro.NewAgent("worker", micro.AgentMaxSteps(8))
```

### Stop on repeat — `LoopLimit`

Bounds how many times the agent may call the **same tool with the same arguments** in one `Ask`. Identical repeated calls make no progress — `MaxSteps` only bounds them by total count, and a circuit breaker only catches *failures*, not a call that succeeds and is pointlessly repeated. When the limit is hit, the call is refused with a message that tells the model it's looping, so it changes approach instead of spinning:

> loop detected: you have already called "search.Search.Query" with the same arguments 3 times and the result will not change. Stop repeating it — try a different approach, or finish with what you have.

```go
micro.NewAgent("worker", micro.AgentLoopLimit(3))
```

`LoopLimit` is **on by default** (a lenient 3) because identical repeated calls are never useful. Set `AgentLoopLimit(0)` to disable it.

### Gate the action — `ApproveTool`

A hook called before each action runs. Return `false` to block it, with a reason that's surfaced to the model. Use it for human-in-the-loop approval, spend limits, allow/deny lists, or any policy:

```go
micro.NewAgent("worker", micro.AgentApproveTool(
    func(tool string, input map[string]any) (bool, string) {
        if strings.HasPrefix(tool, "billing_") {
            return false, "billing actions require sign-off"
        }
        return true, ""
    }))
```

## ApproveTool is the integration seam

`ApproveTool` is also where an **external policy engine** plugs in. It sees every tool call before execution and can veto, so you can route decisions to your own rules, a budget service, or a third-party runtime-safety layer — without go-micro depending on it. Orchestration stays in the agent; execution safety stays in the hook. That separation is the whole point: you can swap the safety layer without touching the agent.

## Wrap the whole execution — `WrapTool`

`ApproveTool` is a *before* gate. When you need the full lifecycle — timing, logging, metrics, retries, or inspecting the result — wrap the execution instead. `WrapTool` is the tool-side analogue of go-micro's `client.CallWrapper` and `server.HandlerWrapper`: a wrapper takes the next handler and returns a new one, so code before the `next(...)` call runs *before* the tool, and code after runs *after*.

```go
import "go-micro.dev/v6/ai"

func logging(next ai.ToolHandler) ai.ToolHandler {
    return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
        start := time.Now()
        res := next(ctx, call)
        log.Printf("id=%s tool=%s took=%s", call.ID, call.Name, time.Since(start))
        return res
    }
}

micro.NewAgent("worker", micro.AgentWrapTool(logging))
```

The handler signature is the same one every provider uses to execute a tool, and it mirrors a service handler — context first, the call in, a result out:

```go
type ToolHandler func(ctx context.Context, call ToolCall) ToolResult
type ToolWrapper func(ToolHandler) ToolHandler
```

`call.ID` is a correlation ID carried through from the provider, so a wrapper can tie a tool call back to the request it came from. `call.Scan(&v)` decodes the arguments into a typed struct when you'd rather not work with the raw map.

Wrappers run **outside** the built-in guardrails, so they observe every call and its result — including a guardrail's refusal. Multiple wrappers compose outermost-first (the first registered is the outer layer). A "before/after" hook is just the two halves of one wrapper, and retry is calling `next` again — so the wrapper is the single, composable seam for everything around execution, while `MaxSteps`, `LoopLimit`, and `ApproveTool` remain the named guardrails on top of it.

### Reliability metadata

A wrapper has what it needs to build reliability tooling — loop handling, retry policies, auditing — without coupling to the agent:

- **What happened** — a guardrail refusal is tagged with a structured reason on the result, so you switch on it rather than parse a message:

  ```go
  res := next(ctx, call)
  switch res.Refused {
  case ai.RefusedLoop:     // the agent repeated an identical call
  case ai.RefusedMaxSteps: // the step budget was exhausted
  case ai.RefusedApproval: // ApproveTool blocked it
  }
  ```

- **Which run** — `ai.RunInfoFrom(ctx)` returns a correlation id for the run, the agent's name, and the parent run when the call came from a delegated sub-agent:

  ```go
  if run, ok := ai.RunInfoFrom(ctx); ok {
      log.Printf("run=%s parent=%s agent=%s tool=%s", run.RunID, run.ParentID, run.Agent, call.Name)
  }
  ```

- **Per-call detail** — `call.ID` (correlation), `call.Name`; duration is `time.Since(start)` around `next`, and step/attempt counts are naturally counted by the wrapper itself (it sees every call).

## Execution safety at the gateway

When agents reach tools **through the MCP gateway**, the gateway adds its own per-tool policies, independent of the agent:

- **`RateLimit`** — requests-per-second per tool.
- **`CircuitBreaker`** — a tool that fails repeatedly is temporarily blocked, so a failing dependency doesn't cascade.

Together with the agent-side guardrails, that's a full set: bound the count, stop the spin, gate the action, rate-limit and circuit-break at the edge.

## Why it matters for autonomous agents

These are most important when no human is in the loop. An agent [triggered by an event](/blog/21) runs unattended — there's no one to notice it looping or to approve a risky call. The guardrails are what let it fail safely and recover on its own rather than quietly burning resources.

## See also

- [Plan & Delegate](plan-delegate.html) — the agent's built-in tools
- [Agents and Workflows](agents-and-workflows.html) — where agents fit
