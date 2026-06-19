---
layout: default
---

# Roadmap

Go Micro is a framework for building **agents and services** in Go. An agent is a distributed system — it discovers services, calls them, holds state, and recovers from failure — so building an agent is building a service. The roadmap has two jobs: make **agentic development** excellent, and make the **developer experience** around it excellent. Nothing else.

## Where we are (v6)

The foundation is in place:

- **Services** — register, discover, RPC, events; every endpoint is automatically an MCP tool.
- **Agents** — a model with memory and tools that manages services, with `plan`, `delegate`, and guardrails (`MaxSteps`, `LoopLimit`, `ApproveTool`) built in, plus tool-execution middleware (`WrapTool`) and run metadata.
- **Flows** — durable, event-driven workflows: ordered steps that checkpoint and resume after a crash.
- **Interop** — the MCP gateway (services as tools) and the A2A gateway (agents as agents, both directions), both generated from the registry; x402 for paid tools.
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

## Next — agentic depth

- **Durable agent loop.** Flows resume; the agent's own loop does not yet. Reuse `Checkpoint` so a long-running agent survives a restart and continues.
- **Streaming.** `ai.Stream` is stubbed across providers; real chat and long-task UX need it, end to end through A2A `message/stream`.
- **Agent observability.** Wire the new `RunInfo` into OpenTelemetry spans so a run — steps, tool calls, delegation — is traceable. This is also what anyone running it in production will need.

## Later

- **Memory management** — summarization and retrieval (RAG) beyond a fixed buffer.
- **Human-in-the-loop** — pause and resume mid-run (`input-required`), beyond the binary `ApproveTool` gate.
- **x402** — conformance against a live facilitator; paid remote tools as agent tools with spend caps.
- **A2A** — streaming, push notifications, multi-turn tasks.

## Developer experience (ongoing)

- **The CLI inner loop** — scaffold → run → chat → inspect (`runs`/`history`) → deploy, made seamless. This is the main lever for "dramatically improve the experience."
- **UI discipline** — keep only high-value, well-built surfaces; trim or cut the rest. The web UI should never be a worse version of the CLI.
- **Examples & a real-world build** — a maintained example that builds something real with the framework, doubling as the 0→hero reference and continuous battle-testing.
- **Docs in lockstep** — the getting-started guide tracks the code on every change.

## How it's sustained

The framework is the product. It's funded by **sponsorship** from the people and companies who run it — not a hosted service, not an enterprise tier, not venture funding. The model is deliberate: keep refining the framework, aligned users adopt and depend on it, and that dependence funds the work. (See [blog/27](/blog/27) for why.)

## Feedback

Open an issue or start a discussion on [GitHub](https://github.com/micro/go-micro), or join the [Discord](https://discord.gg/WeMU5AGxD).
