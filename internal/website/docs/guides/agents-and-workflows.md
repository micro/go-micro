---
layout: default
---

# Agents and Workflows

Go Micro's AI primitives map directly onto the taxonomy in Anthropic's [Building Effective Agents](https://www.anthropic.com/engineering/building-effective-agents). That post draws one distinction that matters:

- **Workflows** — "LLMs and tools orchestrated through **predefined code paths**." Deterministic.
- **Agents** — "LLMs **dynamically direct their own processes** and tool usage." Model-driven.

Go Micro has both, plus the harness they run inside — and expresses them as plain services and tools, with no graph DSL. That's deliberate: the same post advises finding "the simplest solution possible" and being "cautious with frameworks… they obscure the underlying mechanics."

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

### Flow triggers, Agent reasons

A flow doesn't have to do the reasoning itself. Point it at an agent and it becomes a pure trigger — the event fires, the flow renders the prompt, and a registered agent handles it over RPC with its full capabilities (plan, delegate, memory, guardrails):

```go
f := micro.NewFlow("onboard-user",
    micro.FlowTrigger("events.user.created"),
    micro.FlowPrompt("New user {{.Data}} — get them set up."),
    micro.FlowAgent("conductor"),   // the conductor agent reasons; the flow only triggers
)
```

This is the clean seam between the two halves of the taxonomy: the *workflow* (deterministic, event-driven) hands off to the *agent* (dynamic). One engine, two front doors — an event (`flow`) or a conversation (`agent.Ask`).

### Ordered, durable steps

A flow can be a **task made of ordered steps** rather than a single turn — the predefined path made explicit. Each step is checkpointed before and after, so if the process dies mid-run the run **resumes at the step it stopped on**, without re-running the steps that already completed (and already had their side effects). This is durable execution, store-backed by default, with no separate workflow engine.

```go
f := micro.NewFlow("checkout",
    micro.FlowTrigger("events.order.placed"),
    micro.FlowRetry(2),                       // retry each step; per-step override available
    micro.FlowSteps(
        micro.FlowStep{Name: "reserve", Run: micro.FlowCall("inventory", "Inventory.Reserve")},
        micro.FlowStep{Name: "charge",  Run: micro.FlowCall("payment", "Payment.Charge")},
        micro.FlowStep{Name: "welcome", Run: micro.FlowDispatch("comms")}, // hand a step to an agent
    ),
    // Durable by default; point the default store at Postgres/NATS KV to
    // survive a real restart, or plug in Temporal/Restate via Checkpoint.
)
```

A step's action is an RPC (`FlowCall`), an agent hand-off (`FlowDispatch`), one model turn (`FlowLLM`), or any function. `State` carries a typed payload (`Set`/`Scan`) plus a `Stage` marker — the resume point. Runs are retained for success and failure (audit) unless you set `FlowDeleteOnSuccess`. On restart, `f.Pending(ctx)` lists incomplete runs and `f.Resume(ctx, runID)` continues one. See [examples/flow-durable](https://github.com/micro/go-micro/tree/master/examples/flow-durable).

The pluggability is the usual go-micro shape: the built-in `Checkpoint` is store-backed (swap the store backend freely); implement the `Checkpoint` interface to delegate durability to an external engine. Most teams need neither — the default is durable.

## Agent ↔ `agent`

An [`Agent`](plan-delegate.html) is an agent in Anthropic's exact sense: it **directs itself** — plans, calls tools, evaluates results, and decides the next step over many turns, with memory across them. Use it when you want flexibility and model-driven decisions.

```go
a := micro.NewAgent("conductor",
    micro.AgentServices("task"),
    micro.AgentProvider("anthropic"),
)
a.Ask(ctx, "Plan the launch, create the tasks, and have comms notify the owner.")
```

### Long-running memory

Agents use store-backed conversation memory by default, scoped under the agent's
name. That makes short restarts boring: the next `Ask` reloads the retained
history from the same store backend you already use for services and flows.
Long-running agents can also keep model context bounded without losing useful
prior context. If you want retrieval without summaries, enable bounded active
context plus a durable archive of every turn:

```go
a := micro.NewAgent("conductor",
    micro.AgentServices("task"),
    micro.AgentProvider("anthropic"),
    micro.AgentRetrievalMemory(40),        // active messages kept in prompt context
    micro.AgentMemoryRecallLimit(5),       // archived turns recalled per Ask
)
```

`AgentRetrievalMemory(activeLimit)` switches the default memory to a store-backed
retriever. The active conversation is capped at `activeLimit`, every turn is
archived in the same scoped store used by the agent, and future asks inject
matching archived turns ahead of active context. The built-in ranking is
deterministic and credential-free for CI.

When you also want a rolling summary in active context, use compacting memory:

```go
a := micro.NewAgent("conductor",
    micro.AgentServices("task"),
    micro.AgentProvider("anthropic"),
    micro.AgentCompactMemory(40, 12),      // max active messages, recent messages kept verbatim
    micro.AgentMemoryRecallLimit(5),       // compacted turns recalled per Ask
)
```

`AgentCompactMemory(maxMessages, keepRecent)` switches the default memory to a
deterministic compactor. Once active history grows past `maxMessages`, older
turns move into the durable archive, a provider-neutral summary is injected into
active context, and the newest `keepRecent` messages stay verbatim. On future
asks, archived turns whose text matches the current request are recalled ahead of
the active context. Teams that need embeddings or a vector database can still
provide their own `AgentMemory` implementation.

This is harness memory, not prompt-layer orchestration: services remain the
capabilities, agents remain the dynamic decision makers, and flows remain the
durable predefined paths. Compaction only keeps a scheduled or looping agent from
turning every past turn into model context while still letting it remember facts
that matter to the current service → agent → workflow run.

Checkpointed agent runs and compacted memory share the same store-backed shape.
If a provider call fails after the prompt has been recorded, `agent.Resume` uses
the checkpointed run id and does not append that same user turn a second time;
completed tool results and recalled archived memory remain available for the
retry.

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

Anthropic is emphatic that autonomous agents need stopping conditions, human checkpoints, and sandboxed testing. Go Micro's agent has two built-in guardrails, both as plain options:

**Stopping condition** — `MaxSteps` bounds the number of actions an agent may take per `Ask`. Once exceeded, further tool calls are refused and the model is told to stop and summarize.

```go
micro.NewAgent("conductor",
    micro.AgentServices("task"),
    micro.AgentMaxSteps(8),       // at most 8 tool calls per request
)
```

**Human-in-the-loop** — `ApproveTool` gates each action before it runs. Return `false` to block it; the reason is shown to the model so it can adapt. The internal `plan` tool is never gated (it's bookkeeping, not an action).

```go
micro.NewAgent("conductor",
    micro.AgentServices("task"),
    micro.AgentApproveTool(func(tool string, input map[string]any) (bool, string) {
        if strings.HasPrefix(tool, "billing_") {
            return false, "billing actions require human sign-off"
        }
        return true, ""
    }),
)
```

These are harness guardrails, not a separate policy engine — a counter and a callback on the path every tool call already takes. For anything that must be predictable, still prefer a **workflow**, and test agents against the [integration harness](https://github.com/micro/go-micro/tree/master/internal/harness/plan-delegate).

## Why no graph DSL

Anthropic: "be cautious with frameworks… understand the underlying code." Go Micro's answer is that there is no separate framework to understand — the harness is the service runtime. Workflows and agents are services, and tool use is RPC. `plan` and `delegate` are tools, not a graph DSL. The patterns above are code you can read, not a DSL you have to learn. That's the [direction we took going all in on AI](/blog/14).

## See also

- [Building Effective Agents](https://www.anthropic.com/engineering/building-effective-agents) — Anthropic
- [Plan & Delegate](plan-delegate.html) — the agent's built-in tools
- [Agent Integration Patterns](agent-patterns.html) — multi-agent architectures
- [AI Integration](../ai-integration.html) — agents, flows, and the model interface
