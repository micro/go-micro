---
title: "Architecture"
description: "How services, models and agents fit together."
---

Go Micro began as a framework for service communication. RPC clients and servers,
service discovery, codecs, transports, and pub/sub let applications communicate
without tying their business logic to one infrastructure provider. Storage,
configuration, and authentication support that same application boundary.

Agents extend this foundation: a service endpoint can also be a tool. The
services → agents lifecycle describes how these components compose;
it is not a requirement to adopt all of them or a replacement for RPC.

| Component | Responsibility | Public entry point |
|-----------|----------------|--------------------|
| Service | Implement capabilities and own application data | `micro.NewService` |
| Model | Adapt model requests, responses, and tool calls to a provider | `model.Model` |
| Agent | Interpret a request and use tools, memory, and execution controls | `micro.NewAgent` |
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
those calls. CLI development chat and multi-agent routing use this same agent
harness, with memory and plans stored separately for each conversation. Provider-level tool iteration
and agent-level execution control remain separate responsibilities.

See [AI Integration](../ai-integration/index.md),
[guardrails](../guides/agent-guardrails.md), and
[durability and recovery](../guides/durability.md).

## Interop gateways

- **`micro api`** exposes service RPC over HTTP.
- **`micro mcp`** exposes service endpoints as tools for external agents.
- **`micro a2a`** exposes agents through the Agent2Agent protocol.

Gateways derive their service and agent descriptions from registry metadata.
An application can expose the same capabilities to Go clients, agents, and a UI.

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
