# Go Micro [![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/go-micro.dev/v6?tab=doc)

An agent harness and service framework for Go.

Build an agent, give it services as tools, and use them through conversation.
The `micro` CLI lets you start with an idea: describe what you need, generate Go
services, and talk to the running system. You can also write agents and services
directly in Go and run them in your own binaries.

Service discovery, RPC, messaging, and storage connect the pieces. Agents use
service endpoints as tools; workflows coordinate ordered steps with saved
checkpoints. Each component is pluggable.

## Contents

- [Quick Start](#quick-start)
- [Building Agents](#building-agents)
- [Building Services](#building-services)
- [Calling Services](#calling-services)
- [Workflows](#workflows)
- [Multi-Service Projects](#multi-service-projects)
- [Features](#features)
- [AI Providers](#ai-providers)
- [Examples](#examples)
- [CLI](#cli)
- [Community](#community)
- [Docs](#docs)

## Quick Start

Install [Go 1.25 or newer](https://go.dev/doc/install), then the `micro` CLI:

```bash
go install go-micro.dev/v6/cmd/micro@latest
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`. Set a provider API key and
start a conversation in a new directory:

```bash
export OPENAI_API_KEY=your-api-key
mkdir my-app && cd my-app
micro chat --provider openai
```

Describe a capability you want to build, then ask the agent to use it. For example:

```text
Build a notes service that can save, list, and search notes.
Save a note: the launch review is on Friday.
Find my notes about the launch.
```

With no registered agents, `micro chat` uses its built-in development agent.
When a capability is missing, it can generate a service in the current directory,
compile and start it, and discover its endpoints as tools. You can then use that
service in the same conversation. The generated Go source is yours to inspect
and change.

Services started by this chat session stop when you exit. To continue developing
the generated project, run `micro run` from its directory: it starts the services
with hot reload, an API gateway, and a console. Edit the Go code to extend existing
services. If agents are already registered, `micro chat` routes requests to them;
`micro chat <name>` selects a particular agent.

### Start from a prompt

You can also describe the application up front:

```bash
micro run --prompt "a notes service with saving, listing, and search" --provider openai
```

Micro proposes the services for you to review, generates their Go code and an
agent, builds them, and opens a conversation with the running system.

For other providers, see [AI Providers](#ai-providers). For a walkthrough without
an API key, see the [first-agent example](examples/first-agent/).

## Building Agents

Create your own agent when you want to define its instructions, model, and tools
in Go. An agent registers an `Agent.Chat` endpoint and can be called from the CLI,
other services, or another agent.

In a new directory, initialize a module:

```bash
mkdir assistant && cd assistant
go mod init example.com/assistant
go get go-micro.dev/v6
```

Save this as `main.go`:

```go
package main

import (
    "log"
    "os"

    "go-micro.dev/v6"
)

func main() {
    agent := micro.NewAgent("assistant",
        micro.AgentPrompt("You are a helpful assistant. Use your tools to carry out requests."),
        micro.AgentProvider("openai"),
        micro.AgentAPIKey(os.Getenv("OPENAI_API_KEY")),
        micro.AgentServices(), // Start without application service tools.
    )
    if err := agent.Run(); err != nil {
        log.Fatal(err)
    }
}
```

Run it with `go run .`. In another terminal with your provider key exported, talk
to it:

```bash
micro chat assistant --provider openai
```

### Give the agent services as tools

Develop the capabilities your agent needs as Go services, then replace the empty
`micro.AgentServices()` option with their registered names:

```go
micro.AgentServices("notes", "search"),
```

Restart the agent with the updated options and start those services alongside it. Go Micro discovers their endpoints and
makes them available as tools; method descriptions and request fields tell the
model how to call them. The agent can now act on your application through chat.
The `notes` and `search` names above refer to services you create, not built-ins.

Use `micro run` to develop a project containing your services and agent together.
See [Your First Agent](internal/website/content/en/docs/guides/your-first-agent.md)
for a complete service-and-agent implementation. The standalone agent above uses
the services you assign; service generation belongs to the CLI development chat.

You can also call the agent from Go:

```go
resp, err := agent.Ask(ctx, "Find my notes about the launch.")
if err != nil {
    return err
}
fmt.Println(resp.Reply)
```

### Plan & Delegate

Every agent gets two built-in harness capabilities, exposed as tools — no extra setup or separate graph runtime:

- **`plan`** — for multi-step work, the agent records an ordered plan in its store-backed memory and stays oriented across turns.
- **`delegate`** — the agent hands a self-contained subtask to another agent. If a registered agent already owns the relevant services, the hand-off goes over RPC to that agent; otherwise a focused, short-lived sub-agent is created for the subtask with its own isolated context.

This keeps intelligence distributed: an agent doesn't need to know *how* to do everything, only *who* does. See [examples/agent-plan-delegate](examples/agent-plan-delegate/).

```go
// A sub-agent is just an agent — created with New, talked to with Ask.
// delegate-first: reuse a registered agent, or spin up a focused one.
resp, _ := agent.Ask(ctx, "Plan the launch, create the tasks, and have comms notify the owner.")
```

### Batteries included, pluggable

Just as a service composes pluggable abstractions (registry, broker, store), an agent composes a **model**, **memory**, and **tools** — sane defaults out of the box, each swappable.

```go
agent := micro.NewAgent("assistant",
    micro.AgentProvider("anthropic"),                 // model — swap the provider
    micro.AgentCompactMemory(40, 12),                 // memory — durable, summarized, recallable
    micro.AgentTool("weather", "Get the weather for a city",
        map[string]any{"city": map[string]any{"type": "string"}},
        func(ctx context.Context, in map[string]any) (string, error) {
            return getWeather(in["city"].(string))    // tools beyond your services — any function
        }),
    micro.AgentMaxSteps(8),                            // guardrails
)
```

**Memory** is durable and store-backed by default (Postgres, NATS KV, or file), so an agent restores its conversation history after a restart — or supply your own with `AgentMemory`. Long-running agents can opt into `AgentCompactMemory(maxMessages, keepRecent)`: older turns are collapsed into a deterministic summary, recent turns stay verbatim, and relevant archived turns are recalled on future asks without replaying the whole conversation. **Tools** are your services automatically, plus any function you register with `AgentTool`.

Conversation recovery is separate from execution recovery: interrupted agent runs
need an explicit `AgentWithCheckpoint` and resume call. Flow steps resume from
successfully saved boundaries; an interrupted step may execute again. See
[Durability and Recovery](internal/website/content/en/docs/guides/durability.md)
for storage requirements, retry semantics, tool replay, and the exactly-once and
multi-replica limitations.

### Paid tools (x402)

Every endpoint is an AI-callable tool — and it can be a *paid* tool. Go Micro supports [x402](https://x402.org), the HTTP 402 payment standard for agents, so a tool can require a stablecoin payment and an agent can settle it autonomously. It's opt-in and carries no crypto in the framework: verification is delegated to a pluggable facilitator (Coinbase, Alchemy, self-hosted), so Base and Solana are just different facilitators.

See the [Payments (x402) guide](internal/website/content/en/docs/guides/x402-payments.md).

### Reachable by other agents (A2A)

Within a Go Micro system, agents reach each other over RPC. To make them reachable by agents on *other* frameworks, Go Micro speaks the [Agent2Agent (A2A) protocol](https://a2a-protocol.org). The A2A gateway discovers your agents from the registry, generates an Agent Card for each from its metadata — the same way the MCP gateway derives tools from service endpoints — and translates incoming A2A tasks to the agent's `Agent.Chat` RPC. No per-agent code: register an agent and it's reachable over A2A.

An agent can also serve its own A2A endpoint directly:

```go
micro.NewAgent("task-mgr", micro.AgentServices("task"), micro.AgentA2A(":4000"))
```

It works both ways. To call an agent on another framework, an `a2a.Client` is wired into the two places that hand off work: `flow.A2A(url)` as a workflow step (the cross-framework `Dispatch`), and `delegate` to an `http(s)` URL from inside an agent.

MCP exposes your services as tools; A2A exposes your agents as agents. See the [A2A guide](internal/website/content/en/docs/guides/a2a-protocol.md).

## Building Services

Add Go Micro to a Go module:

```bash
mkdir greeter && cd greeter
go mod init example.com/greeter
go get go-micro.dev/v6
```

Save this as `main.go`. A service is a struct with RPC methods:

```go
package main

import (
    "context"
    "log"

    "go-micro.dev/v6"
)

type Request struct {
    Name string `json:"name"`
}

type Response struct {
    Message string `json:"message"`
}

type Say struct{}

// Hello greets a person by name.
// @example {"name": "Alice"}
func (h *Say) Hello(ctx context.Context, req *Request, rsp *Response) error {
    rsp.Message = "Hello " + req.Name
    return nil
}

func main() {
    service := micro.NewService("greeter")
    service.Init()
    if err := service.Handle(new(Say)); err != nil {
        log.Fatal(err)
    }
    if err := service.Run(); err != nil {
        log.Fatal(err)
    }
}
```

Run it with the Go toolchain:

```bash
go run .
```

This starts the service and registers it for discovery. No model provider or API key is needed. Use `go build .` to build a standalone binary. HTTP and MCP gateways can be added when needed.

## Calling Services

Use the Go client to discover and call a running service. In another Go program, create a client service and send a request:

```go
service := micro.NewService("greeter-client")
service.Init()

request := service.Client().NewRequest("greeter", "Say.Hello", &Request{Name: "Alice"})
var response Response
if err := service.Client().Call(ctx, request, &response); err != nil {
    return err
}
fmt.Println(response.Message)
```

Here `Request` and `Response` are the types from the service above, and `ctx` is a `context.Context`. For protobuf contracts and generated clients, see [gRPC interoperability](examples/grpc-interop/).

## Workflows

Define ordered work in Go, with RPC calls and agent dispatch as steps:

```go
service := micro.NewService("fulfillment")
service.Init()
opts := service.Options()
if err := opts.Broker.Connect(); err != nil {
    return err
}
defer opts.Broker.Disconnect()

workflow := micro.NewFlow("order-fulfillment",
    micro.FlowTrigger("orders.created"),
    micro.FlowSteps(
        micro.FlowStep{Name: "reserve", Run: micro.FlowCall("inventory", "Inventory.Reserve")},
        micro.FlowStep{Name: "notify", Run: micro.FlowDispatch("order-assistant")},
    ),
)
if err := workflow.Register(opts.Registry, opts.Broker, service.Client()); err != nil {
    return err
}
defer workflow.Stop()
if err := service.Run(); err != nil {
    return err
}
```

Flows checkpoint between steps. Use persistent storage for recovery across restarts, and make external side effects idempotent because an interrupted step can repeat. See [Agents and Workflows](internal/website/content/en/docs/guides/agents-and-workflows.md) and [Durability and Recovery](internal/website/content/en/docs/guides/durability.md) for execution and resume APIs.

## Multi-Service Projects

Run multiple services together:

```go
users := micro.NewService("users", micro.Address(":9001"))
orders := micro.NewService("orders", micro.Address(":9002"))

users.Handle(new(Users))
orders.Handle(new(Orders))

g := micro.NewGroup(users, orders)
g.Run()
```

## Features

### Framework

| Feature | Details |
|---------|---------|
| Service registry | mDNS (default), Consul, etcd |
| RPC client/server | gRPC transport, load balancing, streaming |
| Pub/sub events | NATS, RabbitMQ, HTTP broker |
| Key-value store | File (bbolt), Postgres, NATS KV |
| Everything swappable | All abstractions are Go interfaces |

### AI

| Feature | Details |
|---------|---------|
| Agents | `micro.NewAgent()` — intelligent layer that manages services |
| Plan & delegate | Built-in agent tools — plan multi-step work, delegate subtasks to other agents |
| Pluggable memory | Durable store-backed conversation memory by default; swap with `AgentMemory` |
| Custom tools | `AgentTool` — give an agent any function as a tool, beyond its services |
| Guardrails | `MaxSteps` (stop on count), `LoopLimit` (stop repeated no-progress calls), `ApproveTool` (human-in-the-loop) |
| Tool middleware | `AgentWrapTool` — wrap tool execution for logging, metrics, or retries (like client/server wrappers) |
| Workflows | `micro.NewFlow()` — event-driven; one step, ordered durable steps, or triggers an agent |
| Durable execution | Flow steps resume from saved boundaries with a persistent store; interrupted steps can repeat, pluggable backend |
| MCP gateway | Every endpoint is an AI tool automatically |
| A2A gateway | Every agent is reachable over the Agent2Agent protocol; cards generated from the registry (`micro a2a`) |
| Payments (x402) | Opt-in per-call payments for tools via the x402 standard; pluggable facilitator (Base, Solana, …) |
| 9 LLM providers | Anthropic, OpenAI, Gemini, Groq, Mistral, Together, Atlas Cloud, MiniMax, Ollama (local + cloud) |

## AI Providers

Swap providers with a single import — same interface everywhere:

| Provider | Default Model |
|----------|---------------|
| Anthropic | `claude-sonnet-4-20250514` |
| OpenAI | `gpt-4o` |
| Google Gemini | `gemini-2.5-flash` |
| Groq | `openai/gpt-oss-120b` |
| Mistral | `mistral-large-latest` |
| Together AI | `meta-llama/Llama-3.3-70B-Instruct-Turbo` |
| Atlas Cloud | `deepseek-ai/DeepSeek-V3-0324` |
| MiniMax | `MiniMax-M3` |
| Ollama | `llama3.2` (local) |

```go
m := model.New("anthropic", model.WithAPIKey(key))
resp, _ := m.Generate(ctx, &model.Request{Prompt: "hello"})
```

## Examples

Start with the [examples index](examples/README.md) for services, agents, and workflows. The [first-agent example](examples/first-agent/) runs with a mock model and needs no API key:

```bash
# From a checkout of this repository
go run ./examples/first-agent
```

- [hello-world](examples/hello-world/) — Basic RPC service
- [multi-service](examples/multi-service/) — Multiple services in one binary
- [mcp](examples/mcp/) — MCP integration with AI agents
- [first-agent](examples/first-agent/) — Smallest provider-free service-backed agent
- [agent-plan-delegate](examples/agent-plan-delegate/) — Agent planning and multi-agent delegation
- [agent-durable](examples/agent-durable/) — Checkpoint and resume an agent run without replaying completed tool side effects
- [grpc-interop](examples/grpc-interop/) — Call go-micro from any gRPC client

See [all examples](examples/README.md).

## CLI

The optional `micro` CLI provides scaffolding, hot reload, gateways, and deployment for Go applications. Use normal `go run` and `go build` commands for the framework itself.

Install the CLI:

```bash
# Binary (no Go required)
curl -fsSL https://go-micro.dev/install.sh | sh

# Or with Go
go install go-micro.dev/v6/cmd/micro@latest
```

If install or `PATH` checks fail, use the [install troubleshooting guide](internal/website/content/en/docs/guides/install-troubleshooting.md).

The [Quick Start](#quick-start) covers conversational development. For the local
development loop:

| Command | Purpose |
|---------|---------|
| `micro chat --provider openai` | Talk to registered agents, or use the development agent when none are registered |
| `micro run --prompt "describe your application" --provider openai` | Review a design, generate services and an agent, then run them |
| `micro run` | Run the current project with hot reload and a console |
| `micro chat assistant --provider openai` | Talk to a specific running agent |
| `micro inspect agent assistant` | Inspect a registered agent's recorded runs |
| `micro new <name>` | Scaffold a service to implement yourself |

Use `micro agent preflight` before running your first agent and `micro agent doctor`
to diagnose a running project. See [Debugging your agent](internal/website/content/en/docs/guides/debugging-agents.md)
for help with tools, provider configuration, and run history.

See the [CLI reference](cmd/micro/README.md) for commands and project configuration.

## Community

Questions, ideas, or just want to build alongside us? [Join the Discord](https://discord.gg/G8Gk5j3uXr).

## Commercial Support

Running Go Micro in production, or building on it and want help? Paid **support, consulting, training, and retainers** are available directly from the maintainer — and they're what keep the project maintained. See [**Support**](SUPPORT.md) for the tiers, or [open a request](https://github.com/micro/go-micro/issues/new?template=commercial_support.md).

## Docs

- [Getting Started](internal/website/content/en/docs/getting-started/index.md)
- [AI Integration](internal/website/content/en/docs/ai-integration/index.md)
- [Your First Agent](internal/website/content/en/docs/guides/your-first-agent.md)
- [0→hero Reference](internal/website/content/en/docs/guides/zero-to-hero.md)
- [Agents and Workflows](internal/website/content/en/docs/guides/agents-and-workflows.md)
- [Agent Design](internal/docs/AGENT_DESIGN.md)
- [Plan & Delegate](internal/website/content/en/docs/guides/plan-delegate.md)
- [Agent Guardrails](internal/website/content/en/docs/guides/agent-guardrails.md)
- [Payments (x402)](internal/website/content/en/docs/guides/x402-payments.md)
- [MCP & AI Agents](internal/website/content/en/docs/mcp.md)
- [Deployment](internal/website/content/en/docs/deployment.md)
- [Plugins](internal/website/content/en/docs/plugins.md)

Package reference: [pkg.go.dev/go-micro.dev/v6](https://pkg.go.dev/go-micro.dev/v6)
