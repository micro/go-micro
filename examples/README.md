# Go Micro Examples

This directory contains runnable examples that take you through the Go Micro
lifecycle: start with a service, expose it as agent-usable capability, then
coordinate work with workflows.

## Quick Start

Each example can be run with `go run .` from its directory unless its README says
otherwise. If you are new to the repo, start with the [examples wayfinding index](./INDEX.md)
or follow the first-agent path below instead of reading the directories alphabetically.

## Recommended first-agent path

This path is the canonical services → agents → workflows route through the examples map. Debugging and observability wayfinding stays nearby once the first run works.

| Step | Start here | What you learn | Next step |
|------|------------|----------------|-----------|
| 1. First service | [`hello-world`](./hello-world/) | Build the 0→1 service path: create and register a basic RPC service, add a handler, call it with a client, and expose health checks. | Move to [`agent-demo`](./agent-demo/) to see services used by an agent. |
| 2. First agent | [`first-agent`](./first-agent/) | Run the smallest service-backed agent with a deterministic mock model and no provider key. | Compare with [`agent-demo`](./agent-demo/) or the maintained 0-to-hero path in [`support`](./support/). |
| 3. First workflow | [`support`](./support/) | Follow typed services into an agent chat loop, an event-driven `intake` flow, and an approval gate in one runnable reference. | Deepen the workflow model with [`flow-durable`](./flow-durable/). |

For the shortest AI-tooling bridge, the MCP path is
[`mcp/hello`](./mcp/hello/) → [`mcp/crud`](./mcp/crud/) →
[`mcp/workflow`](./mcp/workflow/). For debugging and production hardening, keep
[`agent-wrap-tool`](./agent-wrap-tool/), [`agent-durable`](./agent-durable/), and
[`deployment`](./deployment/) nearby.

## Lifecycle map

### 1. Services — learn the runtime foundation

#### [hello-world](./hello-world/)
Basic RPC service demonstrating core concepts:
- Service creation and registration
- Handler implementation
- Client calls
- Health checks

**Run it:**
```bash
cd hello-world
go run .
```

#### [web-service](./web-service/)
HTTP web service with service discovery:
- HTTP handlers
- Service registration
- Health checks
- JSON REST API

**Run it:**
```bash
cd web-service
go run .
```

#### [multi-service](./multi-service/)
Multiple services in a single binary — the modular monolith pattern:
- Isolated server, client, store, and cache per service
- Shared registry and broker for inter-service communication
- Coordinated lifecycle with `service.Group`
- Start monolith, split later when you need to scale independently

**Run it:**
```bash
cd multi-service
go run .
```

#### [deployment](./deployment/)
Docker Compose deployment with MCP gateway, Consul registry, and Jaeger tracing:
- Production-like architecture in one `docker-compose up`
- Standalone MCP gateway connected to service registry
- Distributed tracing with OpenTelemetry + Jaeger

### 2. Agents — turn services into tool-using teammates

#### [first-agent](./first-agent/)
Smallest first agent: one notes service plus one scoped agent, backed by a deterministic mock model so `go run ./examples/first-agent` works without provider secrets.

#### [agent-demo](./agent-demo/)
A multi-service project management app with Projects,
Tasks, and Team services, seed data, and agent playground integration.

#### [agent-plan-delegate](./agent-plan-delegate/)
The two built-in agent capabilities in a small multi-agent system:
- **plan** — an agent records an ordered plan in its store-backed memory before doing multi-step work
- **delegate** — an agent hands a subtask to another agent (over RPC if it's registered, else to an ephemeral sub-agent)

#### [agent-wrap-tool](./agent-wrap-tool/)
Middleware around an agent's tool execution with `AgentWrapTool`, the tool-side analogue of client/server wrappers:
- **observe** — time every tool call and record per-tool metrics, correlated by call ID
- **retry** — re-run a call whose result is an error, recovering from a transient failure before the model sees it

#### [agent-durable](./agent-durable/)
Durable agent runs that can be checkpointed and resumed, useful once your first
agent needs predictable recovery behavior.

#### [agent-human-input](./agent-human-input/)
Human-in-the-loop agent interaction for decisions that need an explicit person
before the run can continue.

#### [agent-ollama](./agent-ollama/)
Local-model agent wiring for developers experimenting with Ollama-backed model
calls.

### 3. Workflows — coordinate longer-running work

#### [support](./support/)
A maintained 0-to-hero reference path in one runnable file:
- **scaffold** typed `customers`, `tickets`, and `notify` services
- **run/chat** with a support agent that uses those services as tools
- **inspect** the event-driven `intake` flow and approval gate
- **CI** keeps the deterministic mock-model journey runnable with `go test ./examples/support`

#### [flow-durable](./flow-durable/)
A workflow as ordered, checkpointed steps that survives a crash and resumes where it stopped:
- **steps** — a flow is a task with stages (`reserve → charge → confirm`), not just one LLM turn
- **Checkpoint** — each step is persisted; on `Resume`, completed steps are not re-run (no duplicate side effects)

#### [flow-loop](./flow-loop/)
A looping flow example for repeated workflow steps.

### 4. MCP and agent integration examples

See the [mcp/](./mcp/) directory for AI agent integration examples:
- **[hello](./mcp/hello/)** - Minimal MCP service (start here)
- **[crud](./mcp/crud/)** - CRUD contact book with full agent documentation
- **[workflow](./mcp/workflow/)** - Cross-service orchestration via AI agents
- **[documented](./mcp/documented/)** - All MCP features with auth scopes
- **[platform](./mcp/platform/)** - Platform-oriented MCP service example

## Other examples

### [auth](./auth/)
Authentication and authorization example.

### [graceful-stop](./graceful-stop/)
Graceful shutdown behavior for long-running services.

### [grpc-interop](./grpc-interop/)
gRPC interoperability example.

## Coming Soon

- **pubsub-events** - Event-driven architecture with NATS
- **grpc-integration** - Using go-micro with gRPC

## Prerequisites

Some examples require external dependencies:

- **NATS**: `docker run -p 4222:4222 nats:latest`
- **Consul**: `docker run -p 8500:8500 consul:latest agent -dev -ui -client=0.0.0.0`
- **Redis**: `docker run -p 6379:6379 redis:latest`

## Contributing

To add a new example:

1. Create a new directory
2. Add a descriptive README.md
3. Include working code with comments
4. Add to this index under the lifecycle stage it supports
5. Ensure it runs with `go run .`
