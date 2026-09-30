---
title: "Architecture"
description: "How services, models, agents, and workflows fit together."
---

Go Micro began as a framework for service communication. RPC clients and servers,
service discovery, codecs, transports, and pub/sub let applications communicate
without tying their business logic to one infrastructure provider. Storage,
configuration, and authentication support that same application boundary.

Agents extend this foundation: a service endpoint can also be a tool. The
services → agents → workflows lifecycle describes how these components compose;
it is not a requirement to adopt all of them or a replacement for RPC.

| Component | Responsibility | Public entry point |
|-----------|----------------|--------------------|
| Service | Implement capabilities and own application data | `micro.NewService` |
| Model | Adapt model requests, responses, and tool calls to a provider | `model.Model` |
| Agent | Interpret a request and use tools, memory, and execution controls | `micro.NewAgent` |
| Workflow | Coordinate predefined steps, with agent decisions where needed | `micro.NewFlow` |
| Application | Combine these components with users, interfaces, and usable outputs | Your program or a runtime such as Mu |

## Service substrate

Services remain useful without a model or agent:

- **Client and server** provide RPC handlers and calls, including streaming.
- **Registry** resolves service names to instances and publishes endpoint metadata.
- **Transport and codec** carry and encode messages. Protobuf contracts and
  generated clients remain supported; reflected Go handlers also expose endpoints
  without generated code.
- **Broker** carries pub/sub events so producers do not need to call consumers.
- **Store** persists records; **config** supplies settings; **auth** supplies
  authentication and authorization abstractions.

These are pluggable Go interfaces. Choosing a registry, broker, or store does not
change a service's business methods. Deployments still need to configure their
backends, permissions, and persistence; the interfaces do not make a deployment
secure or durable automatically.

## Agent harness

The AI packages build on those service contracts:

- **`model` / `model.Model`** adapts provider APIs for generation and streaming.
  Image and video generation have separate interfaces. See the
  [model package](https://github.com/micro/go-micro/tree/master/model) for providers
  and capability reporting.
- **`model.Tools`** discovers endpoint schemas from the registry and invokes them
  through the RPC client. An agent can also register Go functions as custom tools.
- **`agent`** adds instructions, service selection, planning, delegation, tool
  wrappers, limits, and an `Agent.Chat` RPC endpoint.
- **`store` / memory** holds conversation history and run records. Persistent
  memory and resuming interrupted execution are separate concerns; checkpointed
  resume must be configured explicitly.

A request arrives through `Ask`, RPC, or a gateway. The agent presents its tools
to the model, executes requested calls, and returns a reply with tool-call and
run metadata. The service owns the action and its data; the agent chooses when
to use it. Tool selection is not a substitute for service-side authorization.

Built-in providers expose one-turn results to the agent harness, which owns tool
iteration, guardrails, and completion. CLI chat and adaptive flows use that same
harness. Direct v6 `Generate` callers retain legacy provider-owned tool execution
for migration. See the [execution contract](../guides/execution-contract.md) for
strict recovery, provider continuation, and compatibility boundaries.

See [AI Integration](../ai-integration/index.md),
[guardrails](../guides/agent-guardrails.md), and
[durability and recovery](../guides/durability.md).

## Workflows

Use `flow` when the sequence is defined: call a service, validate a result,
dispatch to an agent, and save the outcome. Flows can react to broker events or
run through their execution APIs. A prompt-driven flow can also let a model
choose tools; a workflow does not require every step to use a model.

Flows save checkpoints between steps. Recovery needs persistent storage, and an
interrupted step may execute again. An agent's plan is its working plan;
a workflow is the sequence defined by application code. Neither is automatically
a user-facing task queue or a guarantee that an external action succeeded.

See [Agents and Workflows](../guides/agents-and-workflows.md).

## Interop gateways

- **`micro api`** exposes service RPC over HTTP.
- **`micro mcp`** exposes service endpoints as tools for external agents.
- **`micro a2a`** exposes agents through the Agent2Agent protocol.

Gateways derive their service and agent descriptions from registry metadata.
An application can expose the same capabilities to Go clients, agents, and a UI.

## Services, agents, and flows

Services expose capabilities and own their data. Agents choose capabilities in
response to a goal. Flows coordinate explicit steps, calling services and agents
where needed. The same service remains usable directly, without a model.

The next development priority is consistent execution across these boundaries:
run identity, completion, cancellation, retry budgets, approval, and recovery.
Existing checkpoints are useful, but interrupted operations can execute again;
external effects require service-level idempotency. See the [v7 plan](../v7.md)
for the implementation sequence and acceptance criteria.

Applications can provide interfaces and store outputs using these components.
App generation, rendering, publishing, and product accounts belong to applications
and hosts such as Mu. Go Micro does not define a separate app package.

## Developer path

1. [Quick Start](../quickstart.md): develop services through conversation.
2. [Getting Started](../getting-started/index.md): create a Go agent and choose its tools.
3. [Your First Agent](../guides/your-first-agent.md): run a complete service-backed agent.
4. [Debugging your agent](../guides/debugging-agents.md): inspect tools and run history.

For a provider-free walkthrough, use the
[first-agent example](https://github.com/micro/go-micro/tree/master/examples/first-agent).
The [0→hero Reference](../guides/zero-to-hero.md) contains the maintained lifecycle
checks. See the [API reference](https://pkg.go.dev/go-micro.dev/v6) for individual
interfaces and the [CLI reference](https://github.com/micro/go-micro/blob/master/cmd/micro/README.md)
for project generation and development commands.
