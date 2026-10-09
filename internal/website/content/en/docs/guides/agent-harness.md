---
title: "The Agent Harness"
---

Go Micro's `agent` package combines a model, conversation memory, service tools,
and tool guardrails. `micro chat` uses this package for its development agent and
for routing requests between registered agents. A single registered agent is
called directly and owns its model and credentials.

The framework also supplies discovery, RPC, storage, and flows. Those support
agents, but are not substitutes for managing the model's conversation and tools.

## Current limits

- Development chat saves conversations and plans. `/new` starts a separate
  conversation and `/resume ID` reopens one. Exiting still cancels local work;
  this is conversation persistence, not background execution.
- Generated services become available on the next message. Chat can generate
  services, but does not provide a general file-editing and shell coding agent.
- `micro chat --stream` shows tool events and chunks the completed answer;
  it does not stream provider tokens while the model generates.
- Checkpoints reuse recorded results. They do not make an external action and
  its checkpoint write atomic. An interruption between them can repeat an action;
  services must handle retries safely. See [durability](durability.md).

## The pieces, and what they map to

| Harness concern | In Go Micro | Status |
|---|---|---|
| Tools | Every service endpoint is an MCP-callable tool from registry metadata — no extra code | Shipped |
| Memory | Store-backed agent memory (`AgentMemory`), durable across restarts | Shipped |
| Guardrails | `MaxSteps`, `LoopLimit`, `ApproveTool`, tool wrappers — enforced at the call site | Shipped |
| Workflows | Durable flows; `micro.FlowLoop` for run-until-done | Shipped |
| Planning / delegation | Built-in `plan` and `delegate` tools on every agent | Shipped |
| Discovery & RPC | Registry + client; agents and services find and call each other | Shipped |
| Interop | MCP (tools), A2A (agents), x402 (paid tools) | Shipped |
| Resilience | Per-call timeout with context propagation; opt-in retry/backoff (`ModelRetry`) across the loop | Shipped |
| Durable runs | Checkpoint and resume an agent run with the same checkpoint backend flows use | Shipped |
| Observability | `RunInfo` → OpenTelemetry spans for runs, model calls, tools, delegation, and failures; persisted run history | Shipped |
| Streaming | `model.Stream` through chat, agent, and A2A | In progress |

The "in progress" rows are exactly the roadmap's [Now and Next](/docs/roadmap.html),
and the work is happening in the open.

## Durable agent runs

Agents can persist their execution history to the same `Checkpoint` backend as
flows. A checkpointed `Ask` records the run id, original prompt, model result,
and completed tool calls. If the process restarts after a tool result has been saved but
before the model finishes, `AgentResume` continues the same run and returns the
recorded tool result instead of re-running the side effect. If a run already
completed, resume returns the persisted response without calling the model.

```go
agent := micro.NewAgent("conductor",
    micro.AgentProvider("anthropic"),
    micro.AgentWithCheckpoint(checkpoint),
)

resp, err := agent.Ask(ctx, "charge order 42 and send a receipt")
if err != nil {
    // On startup, or after a transient failure, discover unfinished work:
    pending, _ := micro.AgentPending(ctx, agent)
    for _, run := range pending {
        _, _ = micro.AgentResume(ctx, agent, run.ID)
    }
}
_ = resp
```

Choose the boundary deliberately: use a durable flow when the steps are known
(`reserve`, `charge`, `confirm`) and each step has deterministic retry/resume
semantics. Use a checkpointed agent run when the model is deciding which tools to
call or how many turns it needs, while reusing recorded tool results. Flows and agents share the same
`Checkpoint` interface. Recovery still depends on the service handling repeated
actions safely.

For human-in-the-loop runs that pause through the built-in `request_input` tool,
resume with the operator's response:

```go
_, err := micro.AgentResumeInput(ctx, agent, runID, "Deploy to us-east-1")
```

## Observing agent runs

Pass an OpenTelemetry tracer provider when you construct an agent to turn the
agent's `RunInfo` into spans:

```go
agent := micro.NewAgent("conductor",
    micro.AgentProvider("anthropic"),
    micro.AgentTraceProvider(otel.GetTracerProvider()),
)
```

A traced `Ask` emits a parent `agent.run` span plus child spans for
`agent.model.call` and `agent.tool.call`. Delegate tool calls are marked with
`agent.delegate=true`; ephemeral sub-agents start their own `agent.run` span with
`agent.run.parent_id` set to the delegating run, so a trace shows the hand-off
from service-like agent to sub-agent. Failure and refusal outcomes set error
status on the relevant span and are also recorded in the persisted run timeline.

Important span attributes include:

| Attribute | Meaning |
|---|---|
| `agent.run.id` | Stable run correlation ID surfaced as `model.RunInfo.RunID` |
| `agent.run.parent_id` | Parent run for delegated sub-agent work |
| `agent.name` | Agent that owns the run or call |
| `agent.model.provider` / `agent.model.name` | Provider and configured model for model calls |
| `agent.tool.name` | Tool invoked by the model |
| `agent.delegate` | Whether the tool call is a delegation boundary |
| `agent.latency_ms` | Elapsed time for the run/call |
| `agent.tokens.*` | Token usage when the provider reports it |

## Why services are the right substrate

An agent that does real work needs typed, discoverable, callable capabilities —
which is what a service is. The harness is credible *because* of the service
layer, not in spite of it:

- **Tools are services** — endpoint metadata becomes the tool schema; an RPC
  executes the call.
- **Agents are services** — they register, load-balance, expose `Agent.Chat`, and
  are reachable by other agents.
- **Workflows are code paths** — use a flow when the path is known; hand off to an
  agent when it isn't.
- **Safety lives at execution** — guardrails run on the one path every tool call
  takes.

## When to reach for it

Use Go Micro when the agent has to **operate a system**, not just answer a prompt
— when it needs real tools, state that survives, limits you can enforce, and a way
to be seen and called. If you only need a model in a loop, you don't need a
harness. When that loop has to touch production, you do.

## See also

- [Agents and Workflows](agents-and-workflows.md) — flows vs. agents
- [Agent Loops](agent-loops.md) — run-until-done, with a ceiling
- [Plan & Delegate](plan-delegate.md)
- [Agent Guardrails](agent-guardrails.md)
- [Provider Conformance](provider-conformance.md) — verified provider behavior
- [Roadmap](/docs/roadmap.html)
