---
layout: default
---

# The Agent Harness

The first wave of agent frameworks solved one problem: put a model in a loop with
some tools. The harder problem is **operating** that loop — and that's what a
harness is.

A harness is the runtime around an agent:

- the **tools** it can call,
- the **memory** it keeps,
- the **guardrails** that bound it,
- the **workflows** that trigger and structure it,
- the **state** that survives a restart,
- the **observability** to see what it did,
- the **services** it depends on,
- and the **protocols** other agents use to reach it.

Go Micro's bet is that this runtime is the one you already deploy. An agent is a
service with a model inside; the harness is the distributed-systems machinery
services already have. So you don't bolt a separate orchestration product onto
your stack — the harness *is* the stack.

## The pieces, and what they map to

| Harness concern | In Go Micro | Status |
|---|---|---|
| Tools | Every service endpoint is an MCP-callable tool from registry metadata — no extra code | Shipped |
| Memory | Store-backed agent memory (`AgentMemory`), durable across restarts | Shipped |
| Guardrails | `MaxSteps`, `LoopLimit`, `ApproveTool`, tool wrappers — enforced at the call site | Shipped |
| Workflows | Durable flows; `flow.Loop` for run-until-done | Shipped |
| Planning / delegation | Built-in `plan` and `delegate` tools on every agent | Shipped |
| Discovery & RPC | Registry + client; agents and services find and call each other | Shipped |
| Interop | MCP (tools), A2A (agents), x402 (paid tools) | Shipped |
| Resilience | Per-call timeout with context propagation; opt-in retry/backoff (`ModelRetry`) across the loop | Shipped |
| Durable runs | Checkpoint and resume an agent run with the same checkpoint backend flows use | Shipped |
| Observability | `RunInfo` → OpenTelemetry spans for runs, model calls, tools, delegation, and failures; persisted run history | Shipped |
| Streaming | `ai.Stream` through chat, agent, and A2A | In progress |

The "in progress" rows are exactly the roadmap's [Now and Next](/docs/roadmap.html),
and the work is happening in the open.

## Durable agent runs

Agents can persist their execution history to the same `Checkpoint` backend as
flows. A checkpointed `Ask` records the run id, original prompt, model result,
and completed tool calls. If the process restarts after a tool succeeds but
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
| `agent.run.id` | Stable run correlation ID surfaced as `ai.RunInfo.RunID` |
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

- [Agents and Workflows](agents-and-workflows.html) — flows vs. agents
- [Agent Loops](agent-loops.html) — run-until-done, with a ceiling
- [Plan & Delegate](plan-delegate.html)
- [Agent Guardrails](agent-guardrails.html)
- [Provider Conformance](provider-conformance.html) — verified provider behavior
- [Roadmap](/docs/roadmap.html)
