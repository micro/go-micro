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

To get started follow the getting started guide. 
Otherwise continue to read the docs for more information 
about the framework.

## Contents

- [Getting Started](getting-started.html)
- [MCP & AI Agents](mcp.html) - Turn services into AI-callable tools with the Model Context Protocol
- [CLI & Gateway Guide](guides/cli-gateway.html) - Development vs Production modes
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
- [Contributing](contributing.html)
- [Roadmap](roadmap.html)
