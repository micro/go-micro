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
├── auth/           # Authentication (JWT, no-op)
├── broker/         # Message broker (NATS, RabbitMQ)
├── cache/          # Caching (Redis)
├── client/         # RPC client (gRPC)
├── cmd/micro/      # CLI tool (run, deploy, mcp, build, server)
├── codec/          # Message codecs (JSON, Proto)
├── config/         # Dynamic config (env, file, etcd, NATS)
├── errors/         # Error handling
├── events/         # Event system (NATS JetStream)
├── gateway/
│   ├── api/        # REST API gateway
│   └── mcp/        # MCP gateway (core AI integration)
├── health/         # Health checking
├── logger/         # Logging
├── metadata/       # Context metadata
├── ai/             # AI model providers
│   ├── anthropic/  # Claude provider
│   └── openai/     # GPT provider
├── model/          # Typed data models (CRUD, queries, schemas)
│   ├── memory/     # In-memory backend (dev/testing)
│   ├── sqlite/     # SQLite backend (dev/single-node)
│   └── postgres/   # PostgreSQL backend (production)
├── registry/       # Service discovery (mDNS, Consul, etcd)
├── selector/       # Client-side load balancing
├── server/         # RPC server
├── service/        # Service interface
├── store/          # Data persistence (Postgres, NATS KV)
├── transport/      # Network transport
├── wrapper/        # Middleware (auth, trace, metrics)
├── contrib/        # Community packages
│   └── langchain-go-micro/  # LangChain Python SDK
├── examples/       # Working examples
└── internal/website/docs/   # Documentation site source
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
<<<<<<< claude/changelog-fZd2J
| Roadmap | `ROADMAP_2026.md` |
| Status | `CURRENT_STATUS_SUMMARY.md` |
=======
| Roadmap | `internal/docs/ROADMAP_2026.md` |
| Status | `internal/docs/CURRENT_STATUS_SUMMARY.md` |
>>>>>>> master
| Changelog | `CHANGELOG.md` |
| Docs Site | `internal/website/docs/` |

## Roadmap & Status Documents

- **[ROADMAP.md](ROADMAP.md)** - General framework roadmap
<<<<<<< claude/changelog-fZd2J
- **[ROADMAP_2026.md](ROADMAP_2026.md)** - AI-native era roadmap with business model
- **[CURRENT_STATUS_SUMMARY.md](CURRENT_STATUS_SUMMARY.md)** - Quick status overview
- **[PROJECT_STATUS_2026.md](PROJECT_STATUS_2026.md)** - Detailed technical status
=======
- **[internal/docs/ROADMAP_2026.md](internal/docs/ROADMAP_2026.md)** - AI-native era roadmap with business model
- **[internal/docs/CURRENT_STATUS_SUMMARY.md](internal/docs/CURRENT_STATUS_SUMMARY.md)** - Quick status overview
- **[internal/docs/PROJECT_STATUS_2026.md](internal/docs/PROJECT_STATUS_2026.md)** - Detailed technical status
- **[internal/docs/IMPLEMENTATION_SUMMARY.md](internal/docs/IMPLEMENTATION_SUMMARY.md)** - Implementation notes
>>>>>>> master
- **[CHANGELOG.md](CHANGELOG.md)** - What changed and when

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for full guidelines. Key points:
- Open an issue before large changes
- Include tests for new features
- Run `make test` and `make lint` before submitting
- Follow commit message format: `type: description` (e.g., `feat: add WebSocket transport`)
