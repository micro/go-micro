---
layout: default
---

# Agents and Workflows

Go Micro's AI primitives map directly onto the taxonomy in Anthropic's [Building Effective Agents](https://www.anthropic.com/engineering/building-effective-agents). That post draws one distinction that matters:

- **Workflows** — "LLMs and tools orchestrated through **predefined code paths**." Deterministic.
- **Agents** — "LLMs **dynamically direct their own processes** and tool usage." Model-driven.

Go Micro has both, plus the building block they're made of — and expresses them as plain services and tools, with no graph DSL. That's deliberate: the same post advises finding "the simplest solution possible" and being "cautious with frameworks… they obscure the underlying mechanics."

## The building block: the augmented LLM

Anthropic's foundational unit is the *augmented LLM* — a model with tools, retrieval, and memory. In Go Micro:

| Augmented LLM | Go Micro |
|---|---|
| the model | `ai` package (7 providers, one interface) |
| tools | every service endpoint, discovered from the registry |
| memory | the `store` (file, Postgres, NATS KV) |

Every endpoint is automatically a tool, so the augmented LLM is the default, not something you assemble.

## Workflow ↔ `flow`

A [`Flow`](../ai-integration.html) is a workflow in Anthropic's exact sense: a **predefined path** — an event on a broker topic triggers a prompt with a fixed set of tools, deterministically. Use it when the task is well-defined and you want predictability.

```go
f := micro.NewFlow("onboard-user",
    micro.FlowTrigger("events.user.created"),
    micro.FlowPrompt("New user {{.Data}} — create a workspace and send a welcome email."),
    micro.FlowProvider("anthropic"),
)
```

## Agent ↔ `agent`

An [`Agent`](plan-delegate.html) is an agent in Anthropic's exact sense: it **directs itself** — plans, calls tools, evaluates results, and decides the next step over many turns, with memory across them. Use it when you want flexibility and model-driven decisions.

```go
a := micro.NewAgent("conductor",
    micro.AgentServices("task"),
    micro.AgentProvider("anthropic"),
)
a.Ask(ctx, "Plan the launch, create the tasks, and have comms notify the owner.")
```

## The patterns — most are already here

Anthropic lists five workflow patterns. Go Micro implements the two richest ones natively, as services and tools, and the rest are ordinary compositions:

| Pattern | Go Micro |
|---|---|
| **Routing** — classify input, dispatch to a specialist | `micro chat`'s router — discovers agents, classifies intent, routes over RPC |
| **Orchestrator-workers** — a central LLM breaks down a task, delegates to workers, synthesizes | the `agent` with **`plan`** (break down) + **`delegate`** (hand to workers) + reply (synthesize) — see [Plan & Delegate](plan-delegate.html) |
| **Prompt chaining** — sequential steps | chain flows, or steps in an agent's plan |
| **Parallelization** — independent subtasks at once | Go concurrency + multiple services/agents; fan out with `delegate` |
| **Evaluator-optimizer** — one LLM generates, another critiques in a loop | two agents over RPC (generator + evaluator) |

The orchestrator-workers example is worth calling out: the conductor agent that plans, creates tasks, and delegates the notification to a `comms` agent **is** orchestrator-workers — built without a graph engine. See [examples/agent-plan-delegate](https://github.com/micro/go-micro/tree/master/examples/agent-plan-delegate).

## Choosing

Follow Anthropic's guidance:

- Start with the **augmented LLM** (a single service call through a model). Most tasks need nothing more.
- Reach for a **workflow** (`flow`) when the path is well-defined and you want predictability.
- Reach for an **agent** (`agent`) when the task needs flexibility and model-driven decisions — and accept the higher cost and the need for guardrails.

## Guardrails

Anthropic is emphatic that autonomous agents need stopping conditions, human checkpoints, and sandboxed testing. Go Micro's agent loop has a bounded number of tool rounds; explicit stopping conditions and human-in-the-loop checkpoints are areas of active work. Until then, prefer a **workflow** for anything that must be predictable, and test agents against the [integration harness](https://github.com/micro/go-micro/tree/master/internal/harness/plan-delegate).

## Why no graph DSL

Anthropic: "be cautious with frameworks… understand the underlying code." Go Micro's answer is that there is no separate framework to understand — workflows and agents are services, and tool use is RPC. `plan` and `delegate` are tools, not a harness. The patterns above are code you can read, not a DSL you have to learn. That's the [bet we made going all in on AI](/blog/14).

## See also

- [Building Effective Agents](https://www.anthropic.com/engineering/building-effective-agents) — Anthropic
- [Plan & Delegate](plan-delegate.html) — the agent's built-in tools
- [Agent Integration Patterns](agent-patterns.html) — multi-agent architectures
- [AI Integration](../ai-integration.html) — agents, flows, and the model interface
