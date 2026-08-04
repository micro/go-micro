---
title: Documentation
linkTitle: Docs
description: "Documentation for the Go Micro agent harness and service framework."
menu:
  main:
    weight: 10
    pre: <i class='fa-solid fa-book'></i>
---
## Overview

<img src="/images/generated/architecture.jpg" alt="Go Micro architecture" style="width: 100%; border-radius: 8px; margin-bottom: 1.5rem;" />

Go Micro is an agent harness and service framework for Go. A harness is the runtime around an agent: tools, memory, guardrails, workflows, state, discovery, and interop. Build an agent and it gets a model, memory, tools, planning, delegation, and service discovery; it is reachable over [MCP](https://modelcontextprotocol.io/) and [A2A](https://a2a-protocol.org). Write services and every endpoint becomes an AI-callable tool. Orchestrate the deterministic parts with durable flows. Agents, services, and flows come from the same primitives because an agent is a distributed system, and building one is building a service.

It's built on a pluggable architecture of Go interfaces: service discovery, client/server RPC, pub/sub, plus auth, caching, and storage. Sane defaults out of the box, everything swappable.

## Learn More

Start with [Getting Started](getting-started/index.md) for install and the first local service. Then follow the first-agent on-ramp in the same order as the README: `micro agent demo` for the installed no-secret CLI affordance, `micro agent quickcheck` (or `micro agent debug`) for the short recovery map, `micro examples` for the provider-free examples map, `micro zero-to-hero` for the maintained lifecycle harness, [examples wayfinding index](https://github.com/micro/go-micro/blob/master/examples/INDEX.md) for the runnable examples map, [the smallest first-agent example](https://github.com/micro/go-micro/tree/master/examples/first-agent) for the fastest provider-free run, [the 0→hero support reference](https://github.com/micro/go-micro/tree/master/examples/support) for the full no-secret lifecycle example, [No-secret first-agent transcript](guides/no-secret-first-agent.md) to run a mock-model support agent, [Your First Agent](guides/your-first-agent.md) to build a service-backed agent and talk to it with `micro chat`, [Debugging your agent](guides/debugging-agents.md) to use `micro inspect agent <name>` for runs and memory, and the [0→hero reference path](guides/zero-to-hero.md) to walk the full scaffold → run → chat → inspect → deploy dry-run lifecycle covered by CI.

Otherwise continue to read the docs for more information about the framework.

## Contents

- [Getting Started](getting-started/index.md)
- [0→hero Reference](guides/zero-to-hero.md) - Walk scaffold → run → chat → `micro inspect agent <name>` → deploy dry-run with CI-backed commands
- `micro agent demo` - Show the provider-free first-agent demo command and next docs steps
- `micro agent quickcheck` (alias: `micro agent debug`) - Show the stalled first-agent recovery map before the full debugging guide
- `micro examples` - Show provider-free first-agent examples in copy/paste order
- [Examples wayfinding index](https://github.com/micro/go-micro/blob/master/examples/INDEX.md) - Choose the first-agent, support, and interop examples from one map
- [Smallest first-agent example](https://github.com/micro/go-micro/tree/master/examples/first-agent) - Run one service-backed agent with a deterministic mock model
- [0→hero support reference](https://github.com/micro/go-micro/tree/master/examples/support) - Run the maintained no-secret services → agents → workflows example
- [No-secret first-agent transcript](guides/no-secret-first-agent.md) - Run the first useful agent path without a provider key
- [Your First Agent](guides/your-first-agent.md) - Build a service-backed agent and talk to it with `micro chat`
- [MCP & AI Agents](mcp/index.md) - Turn services into AI-callable tools with the Model Context Protocol
- [CLI & Gateway Guide](guides/cli-gateway.md) - Development vs Production modes
- [`micro loop` quickstart](guides/micro-loop.md) - Scaffold an autonomous CI-gated improvement loop
- [Quick Start](quickstart.md)
- [Architecture](architecture/index.md)
- [Configuration](config/index.md)
- [Registry](interfaces/registry/index.md)
- [Broker](interfaces/broker/index.md)
- [Client/Server](client-server.md)
- [Transport](interfaces/transport/index.md)
- [Store](store.md)
- [Plugins](plugins.md)
- [Examples](examples/)

## Development & Deployment

- [micro run](guides/micro-run.md) - Local development with hot reload, API gateway, and agent playground
- [micro build & deploy](deployment/index.md) - Build binaries and deploy to production
- [micro server](server.md) - Optional production web dashboard with auth

## AI & Agents

- [0→hero Reference](guides/zero-to-hero.md) - Walk scaffold → run → chat → `micro inspect agent <name>` → deploy dry-run with CI-backed commands
- [No-secret first-agent transcript](guides/no-secret-first-agent.md) - Run the first useful agent path without a provider key
- [Your First Agent](guides/your-first-agent.md) - Build a service-backed agent and talk to it with `micro chat`
- [Building AI-Native Services](guides/ai-native-services.md) - End-to-end tutorial for MCP-enabled services
- [MCP Security Guide](guides/mcp-security.md) - Auth, scopes, rate limiting, and audit logging
- [Tool Description Best Practices](guides/tool-descriptions.md) - Writing docs that make agents effective
- [Agent Integration Patterns](guides/agent-patterns.md) - Multi-agent harness patterns and architectures

## Advanced

- [Framework Comparison](guides/comparison.md) - Including Go Micro vs Dapr for agents, services, and workflows
- [Architecture Decisions](architecture/)
- [Real-World Examples](examples/realworld/)
- [Migration Guides](guides/migration/)
- [Observability](observability/index.md)
- [`micro loop` quickstart](guides/micro-loop.md)
- [Contributing](project/contributing.md)
- [Roadmap](roadmap.md)
