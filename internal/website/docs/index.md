---
layout: default
---

# Docs

Documentation for the Go Micro agent harness and service framework.

## Overview

<img src="/images/generated/architecture.jpg" alt="Go Micro architecture" style="width: 100%; border-radius: 8px; margin-bottom: 1.5rem;" />

Go Micro is an agent harness and service framework for Go. A harness is the runtime around an agent: tools, memory, guardrails, workflows, state, discovery, and interop. Build an agent and it gets a model, memory, tools, planning, delegation, and service discovery; it is reachable over [MCP](https://modelcontextprotocol.io/) and [A2A](https://a2a-protocol.org). Write services and every endpoint becomes an AI-callable tool. Orchestrate the deterministic parts with durable flows. Agents, services, and flows come from the same primitives because an agent is a distributed system, and building one is building a service.

It's built on a pluggable architecture of Go interfaces: service discovery, client/server RPC, pub/sub, plus auth, caching, and storage. Sane defaults out of the box, everything swappable.

## Learn More

Start with [Getting Started](getting-started.html) for install and the first local service. Then follow the first-agent on-ramp in the same order as the README: `micro agent demo` for the installed no-secret CLI affordance, `micro examples` for the provider-free examples map, `micro zero-to-hero` for the maintained lifecycle harness, [examples wayfinding index](https://github.com/micro/go-micro/blob/master/examples/INDEX.md) for the runnable examples map, [the smallest first-agent example](https://github.com/micro/go-micro/tree/master/examples/first-agent) for the fastest provider-free run, [the 0→hero support reference](https://github.com/micro/go-micro/tree/master/examples/support) for the full no-secret lifecycle example, [No-secret first-agent transcript](guides/no-secret-first-agent.html) to run a mock-model support agent, [Your First Agent](guides/your-first-agent.html) to build and chat with a service-backed agent, [Debugging your agent](guides/debugging-agents.html) to use `micro inspect agent <name>` for runs and memory, and the [0→hero reference path](guides/zero-to-hero.html) to walk the full scaffold → run → chat → inspect → deploy dry-run lifecycle covered by CI.

Otherwise continue to read the docs for more information about the framework.

## Contents

- [Getting Started](getting-started.html)
- [0→hero Reference](guides/zero-to-hero.html) - Walk scaffold → run → chat → `micro inspect agent <name>` → deploy dry-run with CI-backed commands
- `micro agent demo` - Show the provider-free first-agent demo command and next docs steps
- `micro examples` - Show provider-free first-agent examples in copy/paste order
- [Examples wayfinding index](https://github.com/micro/go-micro/blob/master/examples/INDEX.md) - Choose the first-agent, support, and interop examples from one map
- [Smallest first-agent example](https://github.com/micro/go-micro/tree/master/examples/first-agent) - Run one service-backed agent with a deterministic mock model
- [0→hero support reference](https://github.com/micro/go-micro/tree/master/examples/support) - Run the maintained no-secret services → agents → workflows example
- [No-secret first-agent transcript](guides/no-secret-first-agent.html) - Run the first useful agent path without a provider key
- [Your First Agent](guides/your-first-agent.html) - Build a service-backed agent end to end
- [MCP & AI Agents](mcp.html) - Turn services into AI-callable tools with the Model Context Protocol
- [CLI & Gateway Guide](guides/cli-gateway.html) - Development vs Production modes
- [`micro loop` quickstart](guides/micro-loop.html) - Scaffold an autonomous CI-gated improvement loop
- [Quick Start](quickstart.html)
- [Architecture](architecture.html)
- [Configuration](config.html)
- [Registry](registry.html)
- [Broker](broker.html)
- [Client/Server](client-server.html)
- [Transport](transport.html)
- [Store](store.html)
- [Plugins](plugins.html)
- [Examples](examples/)

## Development & Deployment

- [micro run](guides/micro-run.html) - Local development with hot reload, API gateway, and agent playground
- [micro build & deploy](deployment.html) - Build binaries and deploy to production
- [micro server](server.html) - Optional production web dashboard with auth

## AI & Agents

- [0→hero Reference](guides/zero-to-hero.html) - Walk scaffold → run → chat → `micro inspect agent <name>` → deploy dry-run with CI-backed commands
- [No-secret first-agent transcript](guides/no-secret-first-agent.html) - Run the first useful agent path without a provider key
- [Your First Agent](guides/your-first-agent.html) - Build a service-backed agent end to end
- [Building AI-Native Services](guides/ai-native-services.html) - End-to-end tutorial for MCP-enabled services
- [MCP Security Guide](guides/mcp-security.html) - Auth, scopes, rate limiting, and audit logging
- [Tool Description Best Practices](guides/tool-descriptions.html) - Writing docs that make agents effective
- [Agent Integration Patterns](guides/agent-patterns.html) - Multi-agent harness patterns and architectures

## Advanced

- [Framework Comparison](guides/comparison.html)
- [Architecture Decisions](architecture/)
- [Real-World Examples](examples/realworld/)
- [Migration Guides](guides/migration/)
- [Observability](observability.html)
- [`micro loop` quickstart](guides/micro-loop.html)
- [Contributing](contributing.html)
- [Roadmap](roadmap.html)
