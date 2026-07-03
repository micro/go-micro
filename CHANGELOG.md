# Changelog

All notable changes to Go Micro are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/) and versions
follow [Semantic Versioning](https://semver.org/), matching the git tags and
[GitHub releases](https://github.com/micro/go-micro/releases) (`v6.MINOR.PATCH`).
Patch releases are cut automatically as the loop merges improvements; the
`[Unreleased]` section below is kept current between tags and rolled into the
next version when it ships.

> Earlier `2026.0x` headings are historical calendar-style markers from before
> v6 tagging; they are kept for continuity and not reused.

---

## [Unreleased]

### Added
- **MiniMax provider** — run agents against MiniMax's `MiniMax-M3` model via its OpenAI-compatible endpoint, with tool calling and streaming; auto-detected from the base URL. (`ai/minimax/`)

### Fixed
- **Plan/delegate completion** — agents now continue unfinished plan steps more reliably, fail checkpointed runs that leave delegated plans unfinished, recover from unknown plan-delegate tool calls, avoid duplicate side effects, and complete timeout paths deterministically. (`agent/`)
- **AtlasCloud tool calls** — streaming and request fallback handling now recovers tool-call results from provider responses that omit the expected structured fields. (`ai/atlascloud/`)

---

## [6.3.13] - July 2026

### Added
- **`micro loop`** — scaffold an autonomous improvement loop into any repository: GitHub Actions workflows dispatched to an @mention-driven coding agent, across up to five roles — `planner` (ranked queue), `builder` (top item as a single-concern PR, auto-merged on green CI), `triage` (CI failures → fix issues), and opt-in `coherence` (docs/CHANGELOG alignment) and `release` (daily patch tag). Each dispatch role's instruction lives in an editable `.github/loop/prompts/<role>.md` file — the workflow is the mechanism, the prompt is the policy — so a repo customizes behavior without forking the CLI. `micro loop init --roles …` writes it all; `micro loop verify` checks the wiring. This is the loop that maintains go-micro itself, generalized. (`cmd/micro/loop/`)

### Changed
- **x402 payments** — settlement now covers CDP facilitator authentication and conformance edge cases. (`wrapper/x402/`)

### Fixed
- **Plan/delegate harnessing** — side effects and notifications are now idempotent and deterministic across duplicate, alias, order-scoped, and reachability scenarios. (`agent/`, `internal/harness/`)

### Documentation
- **First-agent on-ramp** — quickstart docs now connect the no-secret first-agent transcript, example map, and 0→hero path. (`README.md`, `internal/website/docs/`)
- **Ollama provider docs** — the provider surface, capability matrix, and examples now document local and cloud behavior. (`internal/website/docs/`, `examples/agent-ollama/`)

---

## [6.3.12] - July 2026

### Added
- **Ollama provider** — run agents against open-weight models locally (`/api/chat`, NDJSON streaming) or via Ollama Cloud (OpenAI-compatible `/v1/chat/completions`, SSE), auto-detected from the base URL, with tool calling in both modes. Point any agent at a non-default endpoint with the new `agent.BaseURL` / `micro.AgentBaseURL` option. (`ai/ollama/`, `examples/agent-ollama/`)
- **Retrieval-backed agent memory** — agents can recall relevant prior turns by similarity, not just the recent window, with a summarizer hook that compacts older history so long conversations stay in budget. (`agent/`)
- **Scheduled flows** — a flow can run an agent (or any step) on a cron-style schedule, with the dispatch traced end to end. (`flow/`)
- **Flow verification/grader loop** — a workflow can grade its own step output against a rubric and retry until it passes, plus run-trace analysis to surface where a flow spends its time. (`flow/`)
- **A2A streaming & continuity** — outbound agent streaming flows through the A2A binding (`message/stream`), with `tasks/resubscribe` and `input-required` handoffs for multi-turn interop. (`gateway/a2a/`)

### Changed
- **Agent tool-call resilience** — opt-in retries around agent tool calls, and a fallback that executes tool calls emitted as text by weaker models so they still make progress. (`agent/`)
- **Hardened agent durability** — terminal failure statuses are classified and surfaced, and durable resume-after-restart is covered by tests. (`agent/`)

### Documentation
- **"Your first agent" walkthrough** and a canonical 0-to-hero reference path, lowering the on-ramp from install to a running agent. (`internal/website/docs/`)
- **Discord** linked prominently across the README, website nav/footer, and docs. (`https://discord.gg/G8Gk5j3uXr`)

---

## [6.0.0] - June 2026

The AI-native major release. Breaking changes are listed first; everything
else is additive. See the [v5 → v6 migration guide](internal/website/docs/guides/migration/v5-to-v6.md) — it's a small upgrade.

### Changed (breaking)
- **Module path is now `go-micro.dev/v6`.** Update imports (`go-micro.dev/v5/...` → `go-micro.dev/v6/...`) and `go install go-micro.dev/v6/cmd/micro@v6`.
- **TLS verification is on by default.** v5 skipped verification unless `MICRO_TLS_SECURE=true`; v6 verifies by default. `MICRO_TLS_SECURE` is removed — set `MICRO_TLS_INSECURE=true` (or call `tls.InsecureConfig()`) for self-signed/dev certs.
- **`micro.NewService(name, opts...)` is the service constructor**, symmetric with `NewAgent`/`NewFlow`. `micro.New(name, opts...)` remains as a deprecated alias; the old name-less `micro.NewService(opts...)` form is removed (pass the name positionally). Generators emit the new form.
- **JWT auth ported in-module.** The external `github.com/micro/plugins/v5/auth/jwt` (pinned to v5) is replaced by `go-micro.dev/v6/auth/jwt/token`, now on the maintained `golang-jwt/jwt/v5`; the deprecated `dgrijalva/jwt-go` dependency is dropped.

### Added
- **A2A protocol — both directions** — `gateway/a2a` exposes registered agents over the open Agent2Agent (A2A) protocol so agents on other frameworks can discover and call them: Agent Cards are generated from registry metadata (the same way the MCP gateway derives tools), and incoming tasks are translated to the agent's existing `Agent.Chat` RPC, with no per-agent code (`micro a2a serve`). The outbound `a2a.Client` calls external A2A agents by URL, wired into `flow.A2A(url)` (a workflow step) and `delegate` to an `http(s)` URL (from inside an agent). An agent can also serve A2A **directly** without a gateway via `AgentA2A(addr)` (`a2a.NewAgentHandler`), handling tasks in-process. The JSON-RPC binding includes `message/send`, `message/stream` (SSE), `tasks/get`, multi-turn continuation by `taskId`/`contextId`, best-effort push notification callbacks, `tasks/resubscribe`, `input-required` handoffs, and card discovery. (`gateway/a2a/`, `cmd/micro/a2a/`)
- **Agents (`micro.NewAgent`)** — an agent is a service with an LLM inside: it discovers its assigned services as tools, runs the model's tool loop, registers a `Chat` RPC endpoint, and is reachable like any service. `Ask` for programmatic use; `micro chat` discovers and routes to agents; `micro agent list`/`describe`. (`agent/`)
- **Plan & delegate** — two built-in agent tools added to every agent: `plan` (an ordered, store-persisted plan surfaced back in the prompt) and `delegate` (hand a self-contained subtask to a registered agent over RPC, otherwise to an ephemeral sub-agent). No harness or graph — they're plain tools. (`agent/builtin.go`, `examples/agent-plan-delegate/`)
- **Agent guardrails** — `MaxSteps` (stop on count), `LoopLimit` (stop repeated no-progress calls; on by default), and `ApproveTool` (human-in-the-loop / policy gate before each action), enforced at the one point every tool call passes through. (`agent/`, guide + blog)
- **Pluggable agent memory & custom tools** — durable store-backed conversation memory by default, swappable via `AgentMemory`; register any function as a tool with `AgentTool`.
- **Workflows (`micro.NewFlow`)** — event-driven orchestration that maps to Anthropic's workflow/agent split: an event triggers a deterministic step (or ordered durable steps), or dispatches to an agent with `FlowAgent`. (`flow/`)
- **Flow loops (`FlowLoop`)** — a flow step that runs a body step repeatedly, carrying state across passes, until a stop condition is met or a hard iteration cap is hit. Stop on a code-defined predicate (`FlowUntil`) or let the model judge it done (`FlowUntilLLM` — the supervised "Ralph" loop); `FlowLoopMax` is the guardrail that guarantees termination, and `FlowOnIteration` reports progress. (`flow/loop.go`, `examples/flow-loop/`, guide)
- **x402 payments** — opt-in per-call payments for tools via the x402 standard, with a pluggable facilitator and a consumer-side client + budget; the MCP gateway can advertise and require payment per tool. (`wrapper/x402/`, guide + blog)
- **Scoped store state** — `store.Scope(s, database, table)` returns a store handle that confines every operation to a database/table without mutating the shared store (unlike `Init(Table(...))`, which is process-global and races between co-located components). Services, agents, and flows now each keep their state in their own table (`service/{name}`, `agent/{name}`, `flow/{name}`); the service path replaces the old `Init(store.Table(name))` global mutation with a scoped handle.
- **Flow discovery & history CLI** — running flows now register in the registry as `type=flow` (and deregister on `Stop`), so they're discoverable like agents: `micro flow list` shows running flows, `micro flow runs <name>` shows a flow's durable run history from the store, and `micro agent history <name>` shows an agent's stored conversation. Live state comes from the registry; durable history from the scoped store.
- **Durable workflows** — a flow can now be an ordered list of steps (a task with stages) that is checkpointed before and after each step, so a run survives a crash and resumes where it stopped without re-running completed steps. State carries a typed payload plus a `Stage` marker; flow-level `Retry` with a per-step override; runs retained for audit unless `DeleteOnSuccess`. Step actions: `Call` (RPC), `LLM` (model turn), `Dispatch` (to an agent), or any `StepFunc`. Durability is a pluggable `Checkpoint` (store-backed by default; implement the interface for Temporal/Restate). Runnable example: `examples/flow-durable/`. Blog: "Durable Workflows" (`internal/website/blog/24.md`).
- **Agent tool-execution wrappers** — `AgentWrapTool` registers middleware around an agent's tool calls, the tool-side analogue of `client.CallWrapper`/`server.HandlerWrapper`. Use it for logging, metrics, retries, or policy; wrappers compose outermost-first and run outside the built-in guardrails. Includes a runnable example with observe + retry wrappers (`examples/agent-wrap-tool/`).
- **Agent platform showcase** — full platform example (Users, Posts, Comments, Mail) mirroring [micro/blog](https://github.com/micro/blog), demonstrating how existing microservices become agent-accessible with zero code changes (`examples/mcp/platform/`).
- **Blog post: "Your Microservices Are Already an AI Platform"** — walkthrough of agent-service interaction patterns using real-world services (`internal/website/blog/7.md`).
- **Circuit breakers for MCP gateway** — per-tool circuit breakers protect downstream services from cascading failures. Configurable max failures, open-state timeout, and half-open probing. Available via `Options.CircuitBreaker` and `--circuit-breaker` CLI flag (`gateway/mcp/circuitbreaker.go`).
- **Helm chart for MCP gateway** — official Helm chart at `deploy/helm/mcp-gateway/` with Deployment, Service, ServiceAccount, HPA, and Ingress templates. Supports Consul/etcd/mDNS registries, JWT auth, rate limiting, audit logging, per-tool scopes, TLS ingress, and auto-scaling.
- **MCP gateway benchmarks** — comprehensive benchmark suite for tool listing, lookup, auth, rate limiting, and JSON serialization (`gateway/mcp/benchmark_test.go`)
- **Workflow example** — cross-service orchestration demo with Inventory, Orders, and Notifications services showing agents chaining multi-step workflows from natural language (`examples/mcp/workflow/`)
- **Docker Compose deployment** — production-like setup with Consul registry, standalone MCP gateway, and Jaeger tracing in one `docker-compose up` (`examples/deployment/`)

---

## [2026.03] - March 2026

### Added

#### Developer Experience
- **`micro new` MCP templates** — `micro new myservice` generates MCP-enabled services with doc comments, `@example` tags, and `WithMCP()` wired in. Use `--no-mcp` to opt out.
- **`micro.NewService("name")` unified API** — single way to create services: `micro.NewService("greeter")` or `micro.NewService("greeter", micro.Address(":8080"))`. Replaces `micro.NewService()` + `service.New()` dual API.
- **`service.Handle()` simplified registration** — register handlers with `service.Handle(new(Greeter))` instead of manual `server.NewHandler` + `server.Handle`.
- **`micro.NewGroup()` modular monoliths** — run multiple services in one binary with shared lifecycle: `micro.NewGroup(users, orders).Run()`.
- **`mcp.WithMCP()` one-liner** — add MCP to any service with a single option: `micro.NewService("name", mcp.WithMCP(":3001"))`.
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
