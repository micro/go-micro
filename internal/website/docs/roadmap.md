---
layout: default
---

# Roadmap

Go Micro is a framework for building **agents and services** in Go. An agent is a distributed system — it discovers services, calls them, holds state, and recovers from failure — so building an agent is building a service. The roadmap has two jobs: make **agentic development** excellent, and make the **developer experience** around it excellent. Nothing else.

## Where we are (v6)

The foundation is in place:

- **Services** — register, discover, RPC, events; every endpoint is automatically an MCP tool.
- **Agents** — a model with memory and tools that manages services, with `plan`, `delegate`, guardrails (`MaxSteps`, `LoopLimit`, `ApproveTool`), tool-execution middleware (`WrapTool`), run metadata, checkpoint/resume, and OpenTelemetry run spans built in.
- **Flows** — durable, event-driven workflows: ordered steps that checkpoint and resume after a crash.
- **Interop** — the MCP gateway (services as tools) and the A2A gateway (agents as agents, both directions, including A2A streaming, push notifications, and multi-turn continuation), both generated from the registry; x402 for paid tools.
- **Secure by default** — TLS verification on, state scoped per component.

## Principles

These constrain everything below:

1. **Build into what people run, never a separate product.** No hosted platform, no enterprise edition. Improvements go deeper into the framework, not beside it.
2. **CLI-first.** The CLI is the experience. Any UI must be genuinely good and earn its place; bloat gets trimmed, not maintained.
3. **The getting-started flow is a contract.** *0→1* (scaffold → run → call) and *0→hero* (the ~10 steps to a working multi-agent system) must always work, and every change is checked against them.
4. **Interaction is as important as running.** Talking to an agent, inspecting runs and history — end to end, not just "it starts."
5. **Battle-tested.** Works across every provider, fails safely, and is observable.

## Now — hardening ("functional on every level")

The priority is that what exists works everywhere, under real conditions.

- **Cross-provider conformance.** Each of the seven providers implements its own tool-call loop; today only the mock and one live provider are exercised. A suite that runs the same agent scenario (tool-calling, multi-step, plan/delegate, guardrails) across every provider, gated on keys, on a schedule.
- **Failure & resilience.** Provider timeouts, rate limits, and cancellation mid-run; deadline/`context` propagation through the agent loop; retry and backoff at the model call.
- **The getting-started contract.** Define and CI-verify the 0→1 and 0→hero flows so they can't silently break.

## Shipped agent depth

- **Durable agent loop.** Opt-in `Checkpoint` support now lets agent `Ask` and
  streaming runs persist, list pending work, and resume without replaying completed
  tool calls. Human-input pauses resume through explicit input helpers.
- **Agent observability.** `RunInfo` now feeds OpenTelemetry spans and events for
  agent runs, model turns, tool calls, retries, delegation lineage, and resume
  checkpoints so production runs are traceable.

## Next — agentic depth

- **Streaming.** Broaden provider-backed `ai.Stream` coverage and keep chat plus A2A `message/stream` working end to end for real chat and long-task UX.
- **Resume operations polish.** Keep improving CLI/docs breadcrumbs for finding
  pending agent runs and deciding whether to call resume, resume-input, or stream
  resume in production.
- **Observability hardening.** Keep span attributes and run inspection coherent
  across agents, flows, and gateways as more providers and workflow paths are
  exercised.

## Later

- **Memory management** — summarization and retrieval (RAG) beyond a fixed buffer.
- **Human-in-the-loop** — broaden pause/resume UX around `input-required` runs and approvals.
- **A2A** — richer live-stream reconnection (`tasks/resubscribe`) and `input-required` handoffs.

## Developer experience (ongoing)

- **The CLI inner loop** — scaffold → run → chat → inspect (`runs`/`history`) → deploy, made seamless. This is the main lever for "dramatically improve the experience."
- **UI discipline** — keep only high-value, well-built surfaces; trim or cut the rest. The web UI should never be a worse version of the CLI.
- **Examples & a real-world build** — a maintained example that builds something real with the framework, doubling as the 0→hero reference and continuous battle-testing.
- **Docs in lockstep** — the getting-started guide tracks the code on every change.

## How it's sustained

The framework is the product. It's funded by **sponsorship** from the people and companies who run it — not a hosted service, not an enterprise tier, not venture funding. The model is deliberate: keep refining the framework, aligned users adopt and depend on it, and that dependence funds the work. (See [blog/27](/blog/27) for why.)

## Feedback

Open an issue or start a discussion on [GitHub](https://github.com/micro/go-micro), or join the [Discord](https://discord.gg/G8Gk5j3uXr).
