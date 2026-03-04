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
├── model/          # AI model providers
│   ├── anthropic/  # Claude provider
│   └── openai/     # GPT provider
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
- **Q2 2026 (Agent DX):** 85% complete (ahead of schedule)
- **Q3 2026 (Production):** 40% complete (ahead of schedule)

### Priority 1: Documentation Guides (HIGHEST ROI)
The framework has features that are under-documented. These guides drive adoption:
1. `docs/guides/ai-native-services.md` - End-to-end tutorial
2. `docs/guides/mcp-security.md` - Auth, scopes, rate limiting for production
3. `docs/guides/tool-descriptions.md` - Writing comments that make agents effective
4. `docs/guides/agent-patterns.md` - Multi-agent workflows and integration patterns

### Priority 2: Multi-Protocol MCP (WebSocket)
Only HTTP/SSE and stdio exist. WebSocket enables bidirectional streaming for real-time agents.

### Priority 3: OpenTelemetry Integration
Trace IDs exist (`Mcp-Trace-Id`) but aren't connected to OTel. This blocks enterprise adoption.

### Priority 4: LlamaIndex SDK
Follow the `contrib/langchain-go-micro/` pattern to build a LlamaIndex integration for RAG.

### Priority 5: Agent Playground Polish
The `/agent` UI in `micro run` needs refinement for demos and onboarding.

## Key Files

| Purpose | File |
|---------|------|
| MCP Gateway | `gateway/mcp/mcp.go` |
| MCP Docs | `gateway/mcp/DOCUMENTATION.md` |
| Model Interface | `model/model.go` |
| CLI Entry | `cmd/micro/main.go` |
| MCP CLI | `cmd/micro/mcp/` |
| Server (run/server) | `cmd/micro/server/server.go` |
| Roadmap | `ROADMAP_2026.md` |
| Status | `CURRENT_STATUS_SUMMARY.md` |
| Docs Site | `internal/website/docs/` |

## Roadmap Documents

- **[ROADMAP.md](ROADMAP.md)** - General framework roadmap
- **[ROADMAP_2026.md](ROADMAP_2026.md)** - AI-native era roadmap with business model
- **[CURRENT_STATUS_SUMMARY.md](CURRENT_STATUS_SUMMARY.md)** - Quick status overview
- **[PROJECT_STATUS_2026.md](PROJECT_STATUS_2026.md)** - Detailed technical status

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for full guidelines. Key points:
- Open an issue before large changes
- Include tests for new features
- Run `make test` and `make lint` before submitting
- Follow commit message format: `type: description` (e.g., `feat: add WebSocket transport`)
