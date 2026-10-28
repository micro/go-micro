---
title: "Roadmap"
---

Go Micro is a framework for building **agents and services** in Go. An agent is a distributed system — it discovers services, calls them, holds state, and recovers from failure — so building an agent is building a service. The roadmap has two jobs: make **agentic development** excellent, and make the **developer experience** around it excellent. Nothing else.

This page is the **canonical roadmap**. `ROADMAP.md` is a repository overview
that links here; when the two differ, this page governs prioritization.

## How roadmap work is tracked

Every accepted roadmap item must have an owning issue, acceptance criteria, a
target release or "no release commitment", a v6-compatible or v7-breaking
classification, and its required tests and documentation. Status is one of
proposed, accepted, in progress, shipped, or deferred.

Issue proposals must also identify effects on service-only applications,
agents, flows, gateways, provider plugins, and stored run/checkpoint formats.
Service-only use remains first-class and must not acquire an AI dependency to
receive framework fixes.

| Workstream | Status | Compatibility | Commitment |
| --- | --- | --- | --- |
| x402 buyer in the agent runtime | Shipped | v6-compatible | v6 |
| AP2 mandate foundation ([#3552](https://github.com/micro/go-micro/issues/3552)) | Proposed | v6-compatible | No release commitment |
| gRPC-reflection MCP | Proposed | v6-compatible | No release commitment |
| Kubernetes operator and CRDs | Proposed | v6-compatible | No release commitment |
| v7 execution contract ([#5044](https://github.com/micro/go-micro/issues/5044)) | Shipped design baseline | mixed; see contract matrix | v6 documentation |
| Restartable service-agent-flow composition ([#5049](https://github.com/micro/go-micro/issues/5049)) | Shipped | v6-compatible fixture | v6 |

## Towards v7 — proposal

A [v7 proposal](v7.md) explores a consistent model/agent execution contract and
an application lifecycle built on services. Work starts with compatible provider
cleanup; an `app` package will be shaped by a complete build, use, revise, and
recover path in Mu and the CLI. This is a design review, not a release commitment.

## Where we are (v6)

The foundation is in place:

- **Services** — register, discover, RPC, events; every endpoint is automatically an MCP tool.
- **Agents** — a model with memory and tools that manages services, with `plan`, `delegate`, guardrails (`MaxSteps`, `LoopLimit`, `ApproveTool`), tool-execution middleware (`WrapTool`), run metadata, checkpoint/resume, and OpenTelemetry run spans built in.
- **Flows** — durable, event-driven workflows: ordered steps that checkpoint and resume after a crash.
- **Interop** — the MCP gateway (services as tools) and the A2A gateway (agents as agents, both directions, including A2A streaming, push notifications, and multi-turn continuation), both generated from the registry; x402 for paid tools.
- **Secure by default** — TLS verification on, state scoped per component.

See [Durability and Recovery](guides/durability.md) for the exact shipped
boundaries: opt-in agent checkpoints, explicit recovery, and no exactly-once
external side-effect or multi-replica lease guarantee.

## Principles

These constrain everything below:

1. **Build into what people run, never a separate product.** No hosted platform, no enterprise edition. Improvements go deeper into the framework, not beside it.
2. **CLI-first.** The CLI is the experience. Any UI must be genuinely good and earn its place; bloat gets trimmed, not maintained.
3. **The getting-started flow is a contract.** *0→1* (scaffold → run → call) and *0→hero* (the ~10 steps to a working multi-agent system) must always work, and every change is checked against them.
4. **Interaction is as important as running.** Talking to an agent, inspecting runs and history — end to end, not just "it starts."
5. **Battle-tested.** Works across every provider, fails safely, and is observable.

## Shipped flagship capability

**Agents that pay (x402 buyer in the runtime).** The opt-in `AgentPayer` and
`AgentBudget` APIs handle payment-required tool results, enforce approval before
payment, settle within a per-run budget, retry the request, and emit spend
events. Key-free tests cover successful payment, retries, missing payers, and
over-budget refusal.

The provider-free `examples/agent-x402-buyer` program documents the complete
local path without live funds. Remaining follow-up is broader interactive
approval UX and provider/gateway conformance when another integration exposes a
gap; it is not grounds to describe the runtime capability as unimplemented.

AP2 mandate work remains a separate proposed foundation and is not a second
flagship milestone.

## Shipped — complete composition

The execution-contract audit, provider-free recovery gate, and single
restartable service-agent-flow composition tracked by
[#5049](https://github.com/micro/go-micro/issues/5049) are shipped. The fixture
calls real services, dispatches bounded agent work through service tools, pauses
for durable approval, and completes one idempotent final operation while
preserving inspectable lineage across replacement hosts. The race-enabled,
key-free fixture is a v6-compatible acceptance gate, not permission to begin
the breaking model/agent split.

The [contract matrix](project/execution-contract.md) remains the compatibility
baseline. gRPC-reflection MCP and Kubernetes deployment remain candidates for
the next net-new capability; neither is accepted as a simultaneous flagship.

## Shipped agent depth

- **Durable agent loop.** Opt-in `Checkpoint` support now lets agent `Ask` and
  streaming runs persist, list pending work, and resume without replaying completed
  tool calls. Human-input pauses resume through explicit input helpers.
- **Agent observability.** `RunInfo` now feeds OpenTelemetry spans and events for
  agent runs, model turns, tool calls, retries, delegation lineage, and resume
  checkpoints so production runs are traceable.

## Next — reach and deployment

- **gRPC-reflection MCP.** Derive MCP tools from external gRPC services through
  server reflection rather than requiring go-micro-native handlers.
- **Kubernetes operator and CRDs.** Reconcile `Agent`, `Service`, and `Flow`
  resources into deployments connected to the registry.

## Later

- **Runtime-fitness loop** — exercise a persistent dogfood application against
  runtime latency, error, and cost signals with canary and rollback boundaries.
- **Memory management** — summarization and retrieval (RAG) beyond a fixed buffer.
- **Human-in-the-loop** — broaden pause/resume UX around `input-required` runs and approvals.
- **A2A** — richer live-stream reconnection (`tasks/resubscribe`) and `input-required` handoffs.

## Ongoing — capped maintenance and developer experience

Cross-provider conformance, cancellation, retry/backoff, streaming,
observability, and the scaffold → run → chat → inspect → deploy loop continue,
but do not compete with the flagship capability as equal roadmap priorities.
The 0→1 and 0→hero paths, mock provider conformance, and deterministic harness
are release gates. Live-provider conformance runs on a schedule when credentials
are available.

## Developer experience (ongoing)

- **The CLI inner loop** — scaffold → run → chat → inspect (`runs`/`history`) → deploy, made seamless. This is the main lever for "dramatically improve the experience."
- **UI discipline** — keep only high-value, well-built surfaces; trim or cut the rest. The web UI should never be a worse version of the CLI.
- **Examples & a real-world build** — a maintained example that builds something real with the framework, doubling as the 0→hero reference and continuous battle-testing.
- **Docs in lockstep** — the getting-started guide tracks the code on every change.

## How it's sustained

The framework is the product. It's funded by **sponsorship** from the people and companies who run it — not a hosted service, not an enterprise tier, not venture funding. The model is deliberate: keep refining the framework, aligned users adopt and depend on it, and that dependence funds the work. (See [Bringing an OSS project back from the dead](/blog/2026/06/18/bringing-an-open-source-project-back-from-the-dead.html) for why.)

See [Adoption and Sustainability](project/adoption.md) for the no-runtime-
telemetry baseline and quarterly aggregate review, and [Version Support](project/version-support.md)
for the release and backport policy.

## Feedback

Open an issue or start a discussion on [GitHub](https://github.com/micro/go-micro), or join the [Discord](https://discord.gg/G8Gk5j3uXr).
