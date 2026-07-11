---
layout: default
---

# Architecture

<img src="/images/generated/architecture.jpg" alt="Go Micro architecture" style="width: 100%; border-radius: 8px; margin: 1rem 0 1.5rem;" />

Go Micro is one runtime for the services → agents → workflows lifecycle. The same
registry, client/server RPC, store, broker, and gateway primitives that run a
service also give an agent discoverable tools, durable state, interop, and a
place to hand off deterministic work.

## Lifecycle map

```text
Services  →  Agents  →  Workflows
handlers     model loop  durable orchestration
registry     memory      triggers and ordered steps
RPC tools    guardrails   agent dispatch
```

The layers are progressive: start with a service, expose its endpoints as tools,
wrap those tools with an agent, then move the known paths into flows so the model
only handles the uncertain parts.

## Service substrate

Go Micro's service framework supplies the distributed-systems base every agent
needs:

- **Registry** — services, agents, and flows register under names so clients,
  gateways, and other agents can discover them without hard-coded addresses. The
  default is mDNS for local development, with pluggable backends for production.
- **RPC client/server** — endpoints are normal Go handlers reached through the
  client, load balanced through discovery, encoded through codecs, and optionally
  streamed.
- **Broker** — asynchronous events connect services and trigger flows without
  coupling producers to consumers.
- **Config and auth** — dynamic configuration plus identity and authorization keep
  local and production runtimes using the same shape.
- **Pluggable interfaces** — registry, broker, store, transport, codecs, auth, and
  config are Go interfaces, so the runtime can stay stable while deployments swap
  infrastructure.

That substrate is intentionally not separate from the agent stack. A service
endpoint is the smallest useful unit of work, and the registry is the source of
truth for which tools and agents exist.

## Agent harness

Agents compose the service substrate with the AI-specific packages:

- **`model` / `ai.Model`** — a pluggable model interface normalizes provider calls
  while letting applications pick Anthropic, OpenAI, Gemini, Atlas Cloud, Groq,
  Mistral, Together AI, or a mock model for no-secret tests.
- **`store` / memory** — agent history, plans, run state, and compacted memory live
  in durable storage rather than in an in-process chat loop.
- **`ai.Tools`** — discovers registered service endpoints and executes them through
  the Go Micro client, so tools are generated from running services instead of a
  parallel tool registry.
- **`agent`** — runs the tool-calling loop with guardrails, planning, delegation,
  service-backed memory, and an `Agent.Chat` RPC endpoint. An agent is therefore a
  service other clients and agents can call.

The result is a harness, not just a prompt loop: model calls are bounded by tool
scope, state is recoverable, and the same CLI and gateways that reach services can
reach agents.

## Workflows

Use `flow` when the path is known or must be repeatable. Flows subscribe to broker
events, run ordered deterministic steps, and can dispatch to an agent at the point
where judgment or language understanding is needed. This keeps long-running work
observable and restartable while preserving agents for open-ended decisions.

A common shape is:

1. A service emits an event such as `ticket.created`.
2. A flow validates and enriches the event with deterministic handlers.
3. The flow dispatches to an agent for classification, drafting, or escalation.
4. The agent calls registered service tools and returns to the flow for final
   durable steps.

## Interop gateways

Gateways project the same runtime to external callers:

- **`micro api`** exposes service RPC over HTTP.
- **`micro mcp`** exposes registered service endpoints as Model Context Protocol
  tools for external agents.
- **`micro a2a`** exposes registered Go Micro agents through the Agent2Agent
  protocol and lets Go Micro flows or agents dispatch to agents hosted elsewhere.

MCP is the services-as-tools boundary; A2A is the agents-as-agents boundary. Both
come from registry metadata, so adding a service or agent updates the external
surface without duplicate wiring.

## Developer path

If you are new, follow the architecture in the same order the runtime composes it:

1. [Install troubleshooting](guides/install-troubleshooting.html) — make sure the
   CLI, `PATH`, version, and no-secret smoke path are healthy.
2. [`micro agent demo`](getting-started.html#first-agent-on-ramp) — print the
   provider-free first-agent command and next docs steps from the installed CLI.
3. `micro agent quickcheck` (or `micro agent debug`) — print the short recovery
   map when scaffold → run → chat → inspect stalls.
4. `micro examples` — list the maintained provider-free runnable examples in
   copy/paste order.
5. `micro zero-to-hero` — print the maintained one-command no-secret lifecycle
   harness and runnable examples.
6. [Examples wayfinding index](https://github.com/micro/go-micro/blob/master/examples/INDEX.md)
   — choose the smallest no-secret first-agent, support reference, and interop
   examples from one map.
7. [Smallest first-agent example](https://github.com/micro/go-micro/tree/master/examples/first-agent)
   — run one service-backed agent with a mock model.
8. [No-secret first-agent transcript](guides/no-secret-first-agent.html) — see the
   maintained support-agent path work without a provider key.
9. [Your First Agent](guides/your-first-agent.html) — build and chat with a
   service-backed agent.
10. [Debugging your agent](guides/debugging-agents.html) — inspect service
    registration, tools, memory, providers, and run history.
11. [0→hero Reference](guides/zero-to-hero.html) — walk scaffold → run → chat →
    inspect → flow → deploy dry-run as the maintained lifecycle contract.

## Related

- [AI Integration](ai-integration.html) — layer-by-layer services → agents → workflows wiring
- [Getting Started](getting-started.html) — first service and first-agent on-ramp
- [Examples](examples/) — runnable examples mapped to the lifecycle
- [ADR Index](architecture/index.md) — architecture decision records
- [Configuration](config.html)
- [Plugins](plugins.html)
