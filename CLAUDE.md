# CLAUDE.md - Go Micro Project Guide

## Project Overview

Go Micro is a framework for distributed systems development in Go. It provides pluggable abstractions for service discovery, RPC, pub/sub, config, auth, storage, and more.

The framework is evolving into an **AI-native platform** where every microservice is automatically accessible to AI agents via the Model Context Protocol (MCP).

## Build & Test

```bash
# Run all tests
make test

# Run tests for a specific package
go test ./gateway/mcp/...
go test ./ai/...
go test ./model/...

# Lint
make lint

# Format
make fmt

# Build CLI
go build -o micro ./cmd/micro

# Run locally with hot reload
micro run
```

## Project Structure

```
go-micro/
├── agent/          # Agent abstraction (intelligent service management)
├── ai/             # AI model providers (Anthropic, OpenAI, Gemini, etc.)
├── auth/           # Authentication (JWT, no-op)
├── broker/         # Message broker (NATS, RabbitMQ)
├── cache/          # Caching (Redis)
├── client/         # RPC client (gRPC)
├── cmd/micro/      # CLI tool (run, deploy, mcp, build, server)
├── codec/          # Message codecs (JSON, Proto)
├── config/         # Dynamic config (env, file, etcd, NATS)
├── errors/         # Error handling
├── events/         # Event system (NATS JetStream)
├── flow/           # Event-driven LLM orchestration
├── gateway/
│   ├── api/        # REST API gateway
│   └── mcp/        # MCP gateway (core AI integration)
│       └── deploy/ # Helm charts for MCP gateway
├── health/         # Health checking
├── logger/         # Logging
├── metadata/       # Context metadata
├── model/          # Typed data models (CRUD, queries, schemas)
├── registry/       # Service discovery (mDNS, Consul, etcd)
├── selector/       # Client-side load balancing
├── server/         # RPC server
├── service/        # Service interface + profiles
├── store/          # Data persistence (Postgres, NATS KV)
├── transport/      # Network transport
├── wrapper/        # Middleware (auth, trace, metrics)
├── examples/       # Working examples
└── internal/       # Non-public: docs, utils, test harness
```

## Key Architectural Decisions

- **Plugin architecture**: All abstractions use Go interfaces. Defaults work out of the box, everything is swappable.
- **Progressive complexity**: Zero-config for development, full control for production.
- **AI-native by default**: Every service is automatically an MCP tool. No extra code needed.
- **In-repo plugins**: Plugins live in the main repo to avoid version compatibility issues.
- **Reflection-based registration**: Handlers are registered via reflection for minimal boilerplate.

## Code Conventions

- Standard Go conventions (gofmt, golint)
- Functional options pattern for configuration (`WithX()` functions)
- Interface-first design: define the interface, then implement
- Tests alongside code (not in separate test directories)
- Commit messages: imperative mood, concise summary line

## Current Focus & Priorities (March 2026)

### Status
- **Q1 2026 (MCP Foundation):** COMPLETE
- **Q2 2026 (Agent DX):** COMPLETE (100%)
- **Q3 2026 (Production):** 50% complete (ahead of schedule)

### Priority 1: Agent Showcase & Examples
Build compelling demos showing agents interacting with go-micro services in realistic scenarios.

### Priority 2: Additional Protocol Support
- gRPC reflection-based MCP
- HTTP/3 support

### Priority 3: Kubernetes & Deployment
- Helm Charts for MCP gateway
- Kubernetes Operator with CRDs

### Recently Completed
- **Agent Plan & Delegate** - Two built-in agent tools: `plan` (ordered plan persisted to store-backed memory, surfaced in the prompt) and `delegate` (hand a subtask to another agent — RPC to a registered agent, else an ephemeral sub-agent with isolated context). Added automatically to every agent; no harness or graph. (`agent/builtin.go`, `examples/agent-plan-delegate/`)
- **`micro new` MCP Templates** - Scaffolds MCP-enabled services with doc comments, `@example` tags, `WithMCP()`. `--no-mcp` to opt out.
- **CRUD Example** - Contact book service with 6 operations, rich agent docs (`examples/mcp/crud/`)
- **Migration Guide** - "Add MCP to Existing Services" guide with 3 approaches
- **Troubleshooting Guide** - Common MCP issues and solutions
- **Error Handling Guide** - Patterns for agent-friendly error responses
- **Documentation Guides** - Six guides: AI-native services, MCP security, tool descriptions, agent patterns, error handling, troubleshooting
- **WithMCP Option** - One-line MCP setup (`gateway/mcp/option.go`)
- **Agent Playground Redesign** - Chat-focused UI with collapsible tool calls
- **Standalone Gateway Binary** - `micro-mcp-gateway` with Docker support
- **WebSocket Transport** - Bidirectional JSON-RPC 2.0 streaming (`gateway/mcp/websocket.go`)
- **OpenTelemetry Integration** - Full span instrumentation with W3C trace context (`gateway/mcp/otel.go`)
- **LlamaIndex SDK** - Python package with RAG examples (`contrib/go-micro-llamaindex/`)

## Key Files

| Purpose | File |
|---------|------|
| MCP Gateway | `gateway/mcp/mcp.go` |
| MCP Docs | `gateway/mcp/DOCUMENTATION.md` |
| AI Interface | `ai/model.go` |
| Model Layer | `model/model.go` |
| CLI Entry | `cmd/micro/main.go` |
| MCP CLI | `cmd/micro/mcp/` |
| Server (run/server) | `cmd/micro/server/server.go` |
| Roadmap | `ROADMAP.md` (full: `internal/website/docs/roadmap.md`) |
| Status | `CHANGELOG.md` |
| Changelog | `CHANGELOG.md` |
| Docs Site | `internal/website/docs/` |

## Roadmap & Status Documents

- **[ROADMAP.md](ROADMAP.md)** - the single, current roadmap (agentic development + DX). Full version at `internal/website/docs/roadmap.md`.
- **[CHANGELOG.md](CHANGELOG.md)** - what shipped and when (the source of truth for status).
- **[internal/docs/IMPLEMENTATION_SUMMARY.md](internal/docs/IMPLEMENTATION_SUMMARY.md)** - Implementation notes
- **[CHANGELOG.md](CHANGELOG.md)** - What changed and when

## Coordination with Codex

Go Micro is maintained by two AI tools — **Claude Code** (you) and **Codex** (its playbook is [CODEX.md](CODEX.md)) — plus the human maintainer, who routes work and owns every merge. To work side by side without collisions:

- **Lanes / branches.** You work on `claude/*` branches; Codex on `codex/*`. Never push to Codex's branch, and never have both agents committing the same branch at once.
- **Base PRs on `master`; don't stack on Codex's in-flight branch.** If that base squash-merges, your commit gets orphaned (this happened — the #3007 fixes had to be re-landed). If the code you need isn't merged yet, wait for it, then branch off `master`. To improve an *open* Codex PR, fix it in place (once Codex is done with the branch, or via an `@codex` comment on the PR) rather than a separate stacked PR.
- **One concern per PR.** Single-purpose PRs; don't bundle (e.g.) a feature with a docs change.
- **Cross-review.** Review Codex's PRs before merge — mechanical fixes you can land yourself (based on `master`), but design/scope/positioning calls go to the human; don't silently rewrite Codex's intent. Codex reviews yours via `@codex review`.
- **Dispatching Codex.** Start a task by commenting `@codex <instruction>` on an issue/PR (that issue/PR is its context). `@codex review` is reserved for review; any other instruction starts a *task*. It's consequential (spends a Codex task slot, pushes commits) and **serial** (one task at a time) — so dispatch one task at a time, only on the human's go-ahead, and never write a literal `@codex` in a comment unless you intend to trigger it (write "Codex" in prose otherwise).
- **CI is the gate.** `go build`, `go test`, `golangci-lint` (blocking), and `make harness` must pass before merge. `internal/harness/` and `examples/` are excluded from errcheck; everything else gets the full set.
- **Backlog = GitHub issues**, each a scoped, self-contained brief with acceptance criteria.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for full guidelines. Key points:
- Open an issue before large changes
- Include tests for new features
- Run `make test` and `make lint` before submitting
- Follow commit message format: `type: description` (e.g., `feat: add WebSocket transport`)
