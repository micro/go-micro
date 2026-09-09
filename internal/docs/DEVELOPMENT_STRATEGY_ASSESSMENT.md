# Development Strategy Assessment

**Date:** 2026-06-24  
**Scope:** README, roadmap, website docs, internal design notes, examples, harnesses, and blog narrative around agents, services, and workflows.

## Executive summary

Go Micro has a coherent v6 thesis: **agents are distributed systems**, so the framework should make agents, services, and workflows one set of Go-native primitives rather than a separate orchestration product. The repository already has the right foundation: services are self-describing tools, agents add model + memory + tools with `plan`/`delegate`, flows provide deterministic event-driven execution, and MCP/A2A/x402 make the system interoperable with external agents and paid tools.

The next phase should not be another broad feature push. The best strategic move is to make the current promise consistently true under real usage:

1. **Harden the core agent loop across providers and failures.**
2. **Make the getting-started contract impossible to break.**
3. **Close the loop on durable, observable, streaming agent runs.**
4. **Turn one maintained real-world build into the canonical 0→hero path.**
5. **Keep the product shape narrow: framework + CLI + docs, not a hosted platform.**

## What the project is now

### Positioning

The public README positions Go Micro as “a framework for building agents and services in Go.” The important distinction is that an agent is not treated as an external chatbot wrapper; it is a service with an LLM, memory, discovered tools, registry presence, and RPC reachability. That makes the messaging clear and differentiated from graph-first agent frameworks.

### Core primitives

- **Services** are the durable base abstraction: ordinary Go handlers register, discover each other, and become AI-callable tools through their endpoint metadata and comments.
- **Agents** are services with a model, memory, and tools. They expose `Agent.Chat`, can be called over RPC, can be reached from `micro chat`, and get built-in `plan` and `delegate` tools.
- **Flows** cover the deterministic side: predefined event or step paths that checkpoint and resume. This complements agents rather than competing with them.
- **Gateways** make the primitives externally useful: MCP exposes services as tools, A2A exposes agents as agents, HTTP/gRPC gateways preserve conventional service access, and x402 opens the path to paid tools.

### Narrative and adoption surface

The blog has progressed from “microservices become AI tools” through first-class agents, planning/delegation, workflows, guardrails, durability, A2A, and sponsorship. That narrative is strong because it tells a product story: Go Micro did not bolt agents onto a framework; it reframed microservices as the runtime substrate for agentic systems.

## Strengths to preserve

1. **One abstraction stack.** Services, agents, and flows all use registry, client, broker, store, and gateway primitives. This lowers conceptual load and reinforces the “distributed systems for agents” thesis.
2. **CLI-first developer experience.** The README and roadmap make the CLI the product surface: scaffold, run, chat, inspect, deploy.
3. **Interop from the registry.** MCP and A2A generated from existing registry metadata avoid hand-maintained tool/agent catalogs.
4. **Pluggable but opinionated.** Model, store, registry, broker, transport, memory, and tool middleware are swappable, but defaults exist.
5. **Good taxonomy.** The docs cleanly map augmented LLMs, workflows, and agents to Go Micro primitives without introducing a graph DSL.

## Key risks

### 1. Breadth outruns reliability

The project already spans services, agents, flows, MCP, A2A, x402, multiple model providers, code generation, CLI, dashboard, examples, and deployment. The roadmap correctly identifies hardening as the current priority. More surface area before conformance and resilience would increase support burden and weaken trust.

### 2. Provider behavior can fragment the experience

The AI package presents one model interface, but each provider has different tool-call semantics, streaming APIs, errors, rate limits, and refusal behavior. If `micro chat`, `NewAgent`, and flows behave differently by provider, users will perceive the framework as unreliable even when the service layer is solid.

### 3. Agent promises require production semantics

Agents that plan, delegate, pay, or run unattended need deadlines, cancellation, retries, resumability, tracing, audit history, and human-intervention states. Without these, agent features remain demos rather than production workflows.

### 4. Docs can become aspirational

Some internal design/status documents are intentionally obsolete or historical. The public roadmap and changelog are now the canonical sources. Future docs should make current/shipped/proposed status explicit to avoid confusion.

### 5. The example portfolio is broad but not yet a single proof

There are many targeted examples, but the strategy needs one polished real-world build that exercises services, agents, flows, guardrails, history/runs, MCP/A2A, and deployment as a continuous path.

## Recommended next steps

### Phase 1: hardening and contracts

**Goal:** make the existing v6 promise repeatable.

1. **Cross-provider conformance matrix**
   - Run the same scenario against every supported provider when keys are present.
   - Cover simple generation, service tool calls, multi-step tool use, `plan`, `delegate`, guardrails, refusal/stop behavior, and structured errors.
   - Publish the support matrix in docs so users know which capabilities are verified.

2. **Getting-started contract in CI**
   - Define two golden paths:
     - **0→1:** `micro new` → `micro run` → HTTP/RPC call succeeds.
     - **0→hero:** services + agent + flow + plan/delegate + inspection path.
   - Make the contract a small harness or scripted example that runs on every relevant change.

3. **Failure semantics in the agent loop**
   - Ensure `context.Context` deadlines propagate through model calls, tool execution, delegation, flows, and gateway calls.
   - Add consistent timeout, retry/backoff, and rate-limit handling at model-provider boundaries.
   - Make cancellation visible in run metadata and user-facing CLI output.

4. **Docs status cleanup**
   - Keep `README.md`, `ROADMAP.md`, and website roadmap aligned.
   - Add or maintain “current vs proposed” banners on internal docs that remain as design history.

### Phase 2: agentic depth

**Goal:** make long-running agent work production-grade.

1. **Durable agent loop**
   - Reuse the existing `Checkpoint` model from flows.
   - Persist model turn, tool call, step count, plan state, delegation context, and terminal status.
   - Resume without duplicating completed side effects.

2. **Agent observability**
   - Convert `RunInfo` into OpenTelemetry spans/events.
   - Include model provider, latency, token usage where available, tool calls, delegate boundaries, refusals, guardrail blocks, and errors.
   - Expose `micro runs` / `micro history` as first-class inspection commands if not already complete.

3. **Streaming end to end**
   - Implement `ai.Stream` uniformly where provider support exists.
   - Carry streaming through `micro chat`, agent `Ask`/`Chat`, A2A `message/stream`, and any UI surface that remains.

4. **Human-in-the-loop state**
   - Extend beyond binary `ApproveTool` into pause/resume or `input-required` states for long runs.
   - Make this compatible with durable checkpoints so approval can happen after process restart.

### Phase 3: canonical real-world build

**Goal:** turn the thesis into one maintained proof path.

Build and maintain one example application that demonstrates:

- A few domain services with real state.
- One conductor agent and at least one specialist agent.
- A flow triggered by an event that dispatches to an agent.
- Guardrails on risky tools.
- Durable run inspection and resume.
- MCP exposure for external agents.
- Optional A2A interop.
- Deployment instructions.

The support-agent example is a strong candidate because it naturally includes service lookups, prioritization, customer communication, human approval, and event-triggered automation.

## Strategic priorities

### Product

- Keep Go Micro as an **open-source framework**, not a hosted platform.
- Make commercial support, training, and retainers the sustainability path.
- Treat CLI quality as the primary adoption lever.

### Developer experience

- Optimize the loop: `new` → `run` → `chat` → `inspect` → `deploy`.
- Prefer fewer, excellent commands over a broad command surface.
- Ensure generated code is ordinary Go that users can edit and keep.

### Technical architecture

- Use registry metadata as the source of truth for tools and agents.
- Keep workflows deterministic and agents dynamic; do not introduce a graph DSL unless the current primitives cannot express a real user need.
- Make all autonomous behavior bounded, observable, cancellable, and resumable.

### Documentation

- Lead with runnable paths and production caveats.
- Keep the taxonomy page because it explains when to use augmented LLMs, flows, and agents.
- Promote one real-world example over many disconnected mini examples.

## Agent harness positioning

The language should move from “framework for services” to “agent harness on top of services.” A service framework helps teams build callable capabilities. An agent harness is the runtime that makes a model safe and useful around those capabilities: discovery, tool schema, execution, state, guardrails, workflows, delegation, observability, and interop.

Recommended public language:

- **Primary line:** “Go Micro is an agent harness and service framework for Go.”
- **Short explanation:** “The harness is the runtime around an agent: tools, memory, guardrails, workflows, state, discovery, and protocols.”
- **Positioning contrast:** “Agent frameworks put a model in a loop; Go Micro operates that loop against real services.”
- **Developer promise:** “If your agent has to operate a system, not just answer a prompt, use Go Micro.”

This keeps the original microservices heritage but reframes it for the agent market. The service layer is not old positioning; it is the reason the harness is credible. Agents need real capabilities, and Go Micro services are typed, discoverable, callable capabilities.

## Relevance in the agent-harness world

To be relevant as agent infrastructure, Go Micro should make the following product bets visible and real:

1. **Harness, not chatbot.** Lead with execution: tools, memory, guardrails, workflows, and interop. Avoid copy that sounds like “another agent framework.”
2. **Services as tools.** Make existing Go services immediately useful to agents through MCP, A2A, and generated tool descriptions.
3. **Runtime safety.** Prioritize MaxSteps, loop detection, approval gates, scoped state, timeouts, cancellation, audit trails, and policy hooks.
4. **Durability and observability.** Agents doing real work need resumable runs, traces, run history, tool-call timelines, and explainable failures.
5. **Interop-first.** Be the Go runtime that any MCP or A2A agent can plug into, rather than a closed agent ecosystem.
6. **Evaluation and conformance.** Harnesses are trusted by tests. Cross-provider conformance, scenario harnesses, and eventually first-class evaluation should become a visible part of the project.
7. **Canonical proof.** Maintain one real-world example that demonstrates services, agents, flows, guardrails, durable runs, MCP/A2A, and deployment end to end.

## Suggested immediate backlog

1. Add cross-provider conformance harness and docs matrix.
2. Script and CI-test the 0→1 and 0→hero getting-started contracts.
3. Audit agent loop context propagation, timeouts, and cancellation.
4. Wire `RunInfo` to tracing and CLI inspection.
5. Implement durable agent checkpoint/resume.
6. Complete streaming through model providers, chat, agent RPC, and A2A.
7. Promote support-agent or another scenario into the canonical real-world example.
8. Normalize docs status banners and remove/redirect stale internal status pages where appropriate.

## Bottom line

Go Micro has a strong, timely strategic position: **the service framework for building agentic systems in Go**. The current opportunity is to make that position trustworthy. Development should bias toward conformance, resilience, durable execution, observability, and a polished end-to-end developer path before expanding the feature surface.
