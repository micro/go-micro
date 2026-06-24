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
| Resilience | Deadlines, timeouts, retry/backoff across the loop | In progress |
| Durable runs | Checkpoint and resume an agent run (flows already do) | In progress |
| Observability | `RunInfo` → OpenTelemetry spans; run history on the CLI | In progress |
| Streaming | `ai.Stream` through chat, agent, and A2A | In progress |

The "in progress" rows are exactly the roadmap's [Now and Next](/docs/roadmap.html),
and the work is happening in the open.

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
- [Roadmap](/docs/roadmap.html)
