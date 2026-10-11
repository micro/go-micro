---
title: Documentation
linkTitle: Docs
description: "Documentation for building services and agents with Go Micro."
menu:
  main:
    weight: 10
    pre: <i class='fa-solid fa-book'></i>
---
## Build services, agents, and workflows

Go Micro connects services through discovery, RPC, and publish/subscribe.
Use it to build services on their own, give agents access to their endpoints as
tools, or coordinate repeatable work with workflows. The CLI provides a local
development experience; the same framework is available to your Go programs.

For how these parts fit together, read the [architecture overview](architecture/index.md).

## Start here

1. [Build your first project](quickstart.md): open `micro chat`, build a service, use it, and extend it.
2. [Keep developing](getting-started/index.md): run your services locally and define your application in Go.
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
