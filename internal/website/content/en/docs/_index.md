---
title: Documentation
linkTitle: Docs
description: "Documentation for the Go Micro agent harness and service framework."
menu:
  main:
    weight: 10
    pre: <i class='fa-solid fa-book'></i>
---
## Build with an agent

Go Micro is an agent harness and service framework for Go. Describe what you need
in `micro chat`, develop services, and use their endpoints as tools through
conversation. You can also create an agent directly with `micro.NewAgent`, choose
its model and instructions, and assign the services it can use.

## Start here

1. [Quick Start](quickstart.md): install the CLI and develop through conversation.
2. [Getting Started](getting-started/index.md): create a Go agent and give it services as tools.
3. [Your First Agent](guides/your-first-agent.md): build and run a complete service-backed agent.

A provider key is needed for live chat and generation. To try the runtime without
one, use the [first-agent example](https://github.com/micro/go-micro/tree/master/examples/first-agent)
or [no-secret transcript](guides/no-secret-first-agent.md).

## Develop and operate

- [AI Integration](ai-integration/index.md): how agents, models, and service tools fit together.
- [micro run](guides/micro-run.md): hot reload, gateway, and interactive console.
- [Debugging your agent](guides/debugging-agents.md): tool calls, memory, and recorded runs.
- [Agents and Workflows](guides/agents-and-workflows.md): combine agent decisions with ordered work.
- [Durability and Recovery](guides/durability.md): persistence, checkpoints, and retries.
- [MCP](mcp.md): expose service endpoints as tools to external clients.
- [A2A](guides/a2a-protocol.md): expose agents to other agent frameworks.
- [Deployment](deployment.md): build binaries and deploy them.
- [CLI & Gateway Guide](guides/cli-gateway.md): local development and production modes.
- [Install troubleshooting](guides/install-troubleshooting.md): installation and PATH help.

## Framework reference

- [Architecture](architecture/index.md)
- [Configuration](config/index.md)
- [Registry](interfaces/registry/index.md)
- [Broker](interfaces/broker/index.md)
- [Client/Server](client-server.md)
- [Transport](interfaces/transport/index.md)
- [Store](store.md)
- [Plugins](plugins.md)
- [Examples](examples/)
- [Go API reference](https://pkg.go.dev/go-micro.dev/v6)

## Optional lifecycle checks

`micro agent demo`, `micro agent quickcheck`, `micro examples`, and
`micro zero-to-hero` print example and troubleshooting commands. The
[0→hero reference](guides/zero-to-hero.md) documents the maintained harness.
