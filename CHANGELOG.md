# Changelog

All notable changes to Go Micro are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/). Go Micro uses
calendar-based versions (YYYY.MM) for the AI-native era.

---

## [Unreleased]

### Added
- **Helm chart for MCP gateway** — official Helm chart at `deploy/helm/mcp-gateway/` with Deployment, Service, ServiceAccount, HPA, and Ingress templates. Supports Consul/etcd/mDNS registries, JWT auth, rate limiting, audit logging, per-tool scopes, TLS ingress, and auto-scaling.
- **MCP gateway benchmarks** — comprehensive benchmark suite for tool listing, lookup, auth, rate limiting, and JSON serialization (`gateway/mcp/benchmark_test.go`)
- **Workflow example** — cross-service orchestration demo with Inventory, Orders, and Notifications services showing agents chaining multi-step workflows from natural language (`examples/mcp/workflow/`)
- **Docker Compose deployment** — production-like setup with Consul registry, standalone MCP gateway, and Jaeger tracing in one `docker-compose up` (`examples/deployment/`)

---

## [2026.03] - March 2026

### Added

#### Developer Experience
- **`micro new` MCP templates** — `micro new myservice` generates MCP-enabled services with doc comments, `@example` tags, and `WithMCP()` wired in. Use `--no-mcp` to opt out.
- **`micro.New("name")` unified API** — single way to create services: `micro.New("greeter")` or `micro.New("greeter", micro.Address(":8080"))`. Replaces `micro.NewService()` + `service.New()` dual API.
- **`service.Handle()` simplified registration** — register handlers with `service.Handle(new(Greeter))` instead of manual `server.NewHandler` + `server.Handle`.
- **`micro.NewGroup()` modular monoliths** — run multiple services in one binary with shared lifecycle: `micro.NewGroup(users, orders).Run()`.
- **`mcp.WithMCP()` one-liner** — add MCP to any service with a single option: `micro.New("name", mcp.WithMCP(":3001"))`.
- **CRUD example** — contact book service with 6 operations, rich agent docs, and validation patterns (`examples/mcp/crud/`).

#### MCP Gateway
- **WebSocket transport** — bidirectional JSON-RPC 2.0 streaming over WebSocket for real-time agent communication (`gateway/mcp/websocket.go`).
- **OpenTelemetry integration** — full span instrumentation across HTTP, stdio, and WebSocket transports with W3C trace context propagation (`gateway/mcp/otel.go`).
- **Standalone gateway binary** — `micro-mcp-gateway` with Docker support for running the MCP gateway independently of services.
- **Per-tool auth scopes** — service-level (`server.WithEndpointScopes()`) and gateway-level (`Options.Scopes`) scope enforcement with bearer token auth.
- **Rate limiting** — per-tool token bucket rate limiting (`Options.RateLimit`).
- **Audit logging** — immutable audit records per tool call with trace ID, account, scopes, duration, and errors (`Options.AuditFunc`).

#### AI Model Package
- **`model.Model` interface** — unified AI provider abstraction with `Generate()` and `Stream()` methods.
- **Anthropic Claude provider** — `model/anthropic` with tool execution and auto-calling.
- **OpenAI GPT provider** — `model/openai` with provider auto-detection from base URL.

#### Agent SDKs
- **LangChain SDK** — `contrib/langchain-go-micro/` Python package with auto-discovery, tool generation, and multi-agent workflow examples.
- **LlamaIndex SDK** — `contrib/go-micro-llamaindex/` Python package with RAG integration examples.

#### Documentation
- **AI-native services guide** — building services for AI agents from scratch
- **MCP security guide** — auth, scopes, and audit logging
- **Tool descriptions guide** — writing doc comments that improve agent performance
- **Agent patterns guide** — architecture patterns for agent integration
- **Error handling guide** — writing agent-friendly error responses with typed errors
- **Troubleshooting guide** — common MCP issues and solutions
- **Migration guide** — add MCP to existing services in 5 minutes

#### CLI
- **`micro mcp serve`** — start MCP server (stdio for Claude Code, HTTP for web agents)
- **`micro mcp list`** — list available tools (human-readable or JSON)
- **`micro mcp test`** — test tools with JSON input
- **`micro mcp docs`** — generate tool documentation
- **`micro mcp export`** — export to LangChain, OpenAPI, or JSON formats

#### Agent Playground
- **Chat-focused UI** — redesigned playground with collapsible tool calls, real-time status, and thinking indicators
- **Provider settings** — configurable OpenAI/Anthropic provider, model, and API key

### Changed
- Service interface moved to `service.Service` with `micro.Service` as a type alias for backward compatibility.
- `service.New()` returns `service.Service` interface (was `*ServiceImpl`).
- `service.NewGroup()` accepts `service.Service` interface (was `*ServiceImpl`).
- `go.mod` template in `micro new` updated to Go 1.22.

### Fixed
- Handler `Handle()` method accepts variadic `server.HandlerOption` for scopes and metadata.
- Store initialization uses service name as table automatically.
- Service `Stop()` properly aggregates errors from lifecycle hooks.

---

## [2026.02] - February 2026

### Added
- **MCP gateway library** — `gateway/mcp/` with HTTP/SSE and stdio transports, service discovery, tool generation, and JSON schema generation from Go types (2,500+ lines).
- **CLI integration** — `micro run --mcp-address` flag to start MCP alongside services.
- **Documentation extraction** — auto-extract tool descriptions from Go doc comments with `@example` tag and struct tag parsing.
- **Blog post** — "Making Microservices AI-Native with MCP"
- **MCP examples** — `examples/mcp/hello/` and `examples/mcp/documented/`

---

## [2026.01] - January 2026

### Added
- **`micro deploy`** — deploy services to any Linux server via SSH + systemd with `micro deploy user@server`.
- **`micro build`** — build Go binaries and Docker images with `micro build --docker`.
- **Blog post** — "Introducing micro deploy"

---

_For earlier changes, see the [git log](https://github.com/micro/go-micro/commits/master)._
