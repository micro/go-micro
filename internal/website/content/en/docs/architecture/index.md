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

There is currently a split in execution responsibility: provider adapters can
perform repeated tool calls inside `Generate` when a tool handler is supplied,
while the agent adds guardrails, run tracking, and plan-completion logic around
those calls. The CLI development chat also has its own session orchestration.
These are separate execution paths, not one universally shared agent loop.

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

## Work, outputs, and apps

A tool can query data, perform an action, or create something that outlives the
conversation. Building an app fits the third case: an app-building service can
own generation, checks, storage, and serving, while an agent calls it as a tool.
The resulting app may itself call services through an authenticated API.

The distinction is between the **run** (the work in progress) and the **output**
(the saved object someone can open, revise, or share). Go Micro currently returns
replies, tool results, and run identifiers; it does not define a common app or
artifact lifecycle. `micro chat` can generate backend services, but that is not
a general UI builder with preview, versioning, and publishing.

[Mu](https://github.com/micro/mu) demonstrates this composition. At the
[reviewed revision](https://github.com/micro/mu/tree/9cdab7dfef90ead71c714cad22d5b99ebd88e937),
its app service exposes build and read methods, keeps durable build jobs, checks
and saves generated HTML, and returns structured output references. Mu's work
layer consumes build events and delivers the result to the conversation. Those
are application capabilities built above Go Micro's services and agent harness.
Mu pins an earlier Go Micro revision, so this demonstrates integration rather
than validation of every feature on the framework's current main branch.

A possible next framework addition is a small, shared output-reference contract:
identity, kind, owning service, revision, and a way to retrieve or open the result,
associated with the run that produced it. This is a design direction, **not an
existing API**. The owning service would still enforce access and decide how to
store, validate, and serve its objects. App rendering, build execution, user
accounts, and publishing policy remain responsibilities of the application or
an optional app service.

Before adding another top-level abstraction, the useful integration target is:
a request creates a saved app, a later request revises that same app, and both
the work status and output remain recoverable after a restart. That exercises
the existing service, agent, and workflow boundaries and makes any missing shared
contract concrete.

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
