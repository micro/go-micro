# Go Micro [![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/go-micro.dev/v6?tab=doc) [![Go Report Card](https://goreportcard.com/badge/github.com/go-micro/go-micro)](https://goreportcard.com/report/github.com/go-micro/go-micro) [![Discord](https://img.shields.io/badge/Discord-join-5865F2?logo=discord&logoColor=white&style=flat-square)](https://discord.gg/G8Gk5j3uXr)

Go Micro is an **agent harness** and service framework for Go.

**Community:** questions, ideas, or just want to build alongside us? [Join the Discord](https://discord.gg/G8Gk5j3uXr).

A harness is the runtime around an agent: the tools it can call, the memory it keeps, the guardrails that bound it, the workflows that trigger it, the services it depends on, and the protocols other agents use to reach it. 

Go Micro gives you the harness as Go code. Build an agent and it gets a model, memory, tools, planning, delegation, guardrails, and service discovery; it is reachable over [MCP](https://modelcontextprotocol.io/) and [A2A](https://a2a-protocol.org). Write services and every endpoint becomes an AI-callable tool. Orchestrate the deterministic parts with durable flows. Agents, services, and flows share one runtime because an agent is a distributed system, and building one is building a service.

## Sponsors

<a href="https://go-micro.dev/blog/3"><img src="https://upload.wikimedia.org/wikipedia/commons/7/78/Anthropic_logo.svg" height="26" /></a>
&nbsp;&nbsp;
<a href="https://go-micro.dev/blog/29"><img src="https://upload.wikimedia.org/wikipedia/commons/4/4d/OpenAI_Logo.svg" height="26" /></a>
&nbsp;&nbsp;
<a href="https://go-micro.dev/blog/8"><img src="https://www.atlascloud.ai/logo.svg" height="26" /></a>

**Want to support Go Micro and see your logo here?** [Become a sponsor](https://discord.gg/G8Gk5j3uXr) — reach out on Discord.

## Commercial Support

Running Go Micro in production, or building on it and want help? Paid **support, consulting, training, and retainers** are available directly from the maintainer — and they're what keep the project maintained. See [**Support**](SUPPORT.md) for the tiers, or [open a request](https://github.com/micro/go-micro/issues/new?template=commercial_support.md).

## Contents

- [Quick Start](#quick-start)
- [Why an Agent Harness](#why-an-agent-harness)
- [Writing Services](#writing-services)
- [Building Agents](#building-agents) — [Plan & Delegate](#plan--delegate), [Pluggable](#batteries-included-pluggable), [Paid tools (x402)](#paid-tools-x402), [A2A](#reachable-by-other-agents-a2a)
- [Features](#features)
- [CLI](#cli)
- [Multi-Service Projects](#multi-service-projects)
- [Data Model](#data-model)
- [AI Providers](#ai-providers)
- [Examples](#examples)
- [Commercial Support](#commercial-support)
- [Docs](#docs)

## Quick Start

Install the CLI:

```bash
# Binary (no Go required)
curl -fsSL https://go-micro.dev/install.sh | sh

# Or with Go
go install go-micro.dev/v6/cmd/micro@latest
```

### Fastest start — no API key

Scaffold a service, run it, call it:

```bash
micro new helloworld
cd helloworld
micro run
```

Then in another terminal:

```bash
curl -X POST http://localhost:8080/api/helloworld/Helloworld.Call \
  -H 'Content-Type: application/json' -d '{"name":"World"}'
```

This install → scaffold → run → call path is covered by no-secret CI harnesses. To
verify just the local installer and first-run CLI boundaries without network
access or provider keys, use:

```bash
make install-smoke
```

To run the broader local contract (including the [0→hero services → agents → workflows path](internal/website/docs/guides/zero-to-hero.md),
chat/inspect CLI boundaries, and deploy dry-run), use:

```bash
make harness
```

### Generate from a prompt — with an LLM key

Set a provider key, describe what you want, and the AI designs services, writes handlers, compiles, and starts them:

```bash
export ANTHROPIC_API_KEY=sk-ant-...   # or OPENAI_API_KEY, GEMINI_API_KEY, ...
micro run --prompt "a task management system with categories" --provider anthropic
```

The AI designs the architecture, you review it, then it generates handlers with real business logic, compiles them, and starts them:

```
Services:
  ● task — Task management with status tracking
  ● project — Project organization

Generate? [Y/n]

Micro
  Services:
    ● task
    ● project
  Agents:
    ◆ agent
```

Then talk to your services from the console:

```
> Create a project called Launch, then add three tasks to it

→ project_Project_Create({"name":"Launch"})
← {"record":{"id":"p1..."},"success":true}
→ task_Task_Create({"title":"Design specs","project_id":"p1..."})
→ task_Task_Create({"title":"Write code","project_id":"p1..."})
→ task_Task_Create({"title":"Ship it","project_id":"p1..."})

Created project Launch and added three tasks to it.
```

When you need a capability that doesn't exist, the agent generates a new service mid-conversation:

```
> I need to track shipping. Create a shipment for order 123 to London.

  ⚡ generating shipping service...
  ✓ shipping
  → shipping_Shipping_Create({"order_id":"123","destination":"London"})
  ← {"record":{"id":"xyz...","status":"pending"}}

  Created shipment for order 123 going to London.
```

Edit the generated code by hand at any time — re-running preserves your changes. [Read more](https://go-micro.dev/blog/13).

## Why an Agent Harness

The first wave of agent frameworks helped developers put a model in a loop. The next problem is operating that loop: connecting it to real tools, scoping what it can touch, preserving state, routing work to specialists, recovering from failures, observing what happened, and letting other agents call it. That is harness work.

Go Micro's answer is to make the harness the same thing you already deploy:

- **Tools are services** — endpoint metadata becomes tool schema; RPC executes the call.
- **Agents are services** — they register, discover, load-balance, and expose `Agent.Chat`.
- **Workflows are durable code paths** — use flows when the path is known; dispatch to agents when it is not.
- **Safety lives at execution** — `MaxSteps`, `LoopLimit`, `ApproveTool`, and tool wrappers run where actions happen.
- **Interop is built in** — MCP for tools, A2A for agents, x402 for paid tools.

Use Go Micro when the agent has to operate a system, not just answer a prompt.

## Writing Services

Under the hood, a service is a struct with methods. Doc comments and `@example` tags become tool descriptions for AI agents automatically.

```go
package main

import (
    "context"

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
    service.Handle(new(Say))
    service.Run()
}
```

Run it and everything is accessible — REST, gRPC, MCP, agent playground:

```bash
micro run
# Dashboard:   http://localhost:8080
# API:         http://localhost:8080/api/{service}/{method}
# Agent:       http://localhost:8080/agent
# MCP Tools:   http://localhost:8080/mcp/tools
```

You can also scaffold a service from a template:

```bash
micro new helloworld
micro new contacts --template crud
```

## Building Agents

An Agent is a service with an LLM inside it. It has a proto-defined `Agent.Chat` RPC endpoint, registers in the registry, and is callable like any service:

```go
agent := micro.NewAgent("task-mgr",
    micro.AgentServices("task", "project"),
    micro.AgentPrompt("You manage tasks and projects. You understand deadlines and priorities."),
    micro.AgentProvider("anthropic"),
)
agent.Run()
```

The agent discovers its services from the registry, scopes its tools to their endpoints, and maintains conversation memory in the store. It registers itself so `micro chat` and other agents can find it.

```go
// Programmatic interaction
resp, _ := agent.Ask(ctx, "What tasks are overdue?")
fmt.Println(resp.Reply)
```

Multiple agents coordinate via RPC — each is a service with an `Agent.Chat` endpoint. `micro chat` routes to the right one.

```bash
micro agent list                    # list registered agents
micro call task-mgr Agent.Chat '{"message": "What tasks are overdue?"}'
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

**Memory** is durable and store-backed by default (Postgres, NATS KV, or file), so an agent picks up where it left off after a restart — or supply your own with `AgentMemory`. Long-running agents can opt into `AgentCompactMemory(maxMessages, keepRecent)`: older turns are collapsed into a deterministic summary, recent turns stay verbatim, and relevant archived turns are recalled on future asks without replaying the whole conversation. **Tools** are your services automatically, plus any function you register with `AgentTool`.

### Paid tools (x402)

Every endpoint is an AI-callable tool — and it can be a *paid* tool. Go Micro supports [x402](https://x402.org), the HTTP 402 payment standard for agents, so a tool can require a stablecoin payment and an agent can settle it autonomously. It's opt-in and carries no crypto in the framework: verification is delegated to a pluggable facilitator (Coinbase, Alchemy, self-hosted), so Base and Solana are just different facilitators.

```bash
# Charge for tool calls at the MCP gateway (off unless you set a pay-to address)
micro mcp serve --x402_pay_to 0xYourAddress --x402_network solana --x402_amount 10000
# Per-tool amounts via a config file
micro mcp serve --x402_config x402.json
```

See the [Payments (x402) guide](internal/website/docs/guides/x402-payments.md).

### Reachable by other agents (A2A)

Within a Go Micro system, agents reach each other over RPC. To make them reachable by agents on *other* frameworks, Go Micro speaks the [Agent2Agent (A2A) protocol](https://a2a-protocol.org). The A2A gateway discovers your agents from the registry, generates an Agent Card for each from its metadata — the same way the MCP gateway derives tools from service endpoints — and translates incoming A2A tasks to the agent's `Agent.Chat` RPC. No per-agent code: register an agent and it's reachable over A2A.

```bash
micro a2a serve --address :4000    # gateway: expose every registered agent over A2A
micro a2a list                     # agents and their Agent Card URLs
```

Or skip the gateway entirely — an agent can serve its own A2A endpoint directly, handling tasks in-process:

```go
micro.NewAgent("task-mgr", micro.AgentServices("task"), micro.AgentA2A(":4000"))
```

It works both ways. To call an agent on another framework, an `a2a.Client` is wired into the two places that hand off work: `flow.A2A(url)` as a workflow step (the cross-framework `Dispatch`), and `delegate` to an `http(s)` URL from inside an agent.

MCP exposes your services as tools; A2A exposes your agents as agents. See the [A2A guide](internal/website/docs/guides/a2a-protocol.md).

## Features

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
| Durable execution | Checkpointed flow steps survive a crash and resume where they stopped; store-backed by default, pluggable backend |
| MCP gateway | Every endpoint is an AI tool automatically |
| A2A gateway | Every agent is reachable over the Agent2Agent protocol; cards generated from the registry (`micro a2a`) |
| Payments (x402) | Opt-in per-call payments for tools via the x402 standard; pluggable facilitator (Base, Solana, …) |
| 8 LLM providers | Anthropic, OpenAI, Gemini, Groq, Mistral, Together, Atlas Cloud, Ollama (local + cloud) |
| Interactive console | `micro run` includes a chat console for talking to services |
| Service generation | `micro run --prompt` — describe a system, get running services |

### Framework

| Feature | Details |
|---------|---------|
| Service registry | mDNS (default), Consul, etcd |
| RPC client/server | gRPC transport, load balancing, streaming |
| Pub/sub events | NATS, RabbitMQ, HTTP broker |
| Key-value store | File (bbolt), Postgres, NATS KV |
| Typed model layer | CRUD + queries, SQLite/Postgres backends |
| Everything swappable | All abstractions are Go interfaces |

### Developer experience & deployment

| Feature | Details |
|---------|---------|
| Hot reload | `micro run` watches files, rebuilds on change |
| Templates | `micro new --template crud/pubsub/api` |
| One-command deploy | `micro deploy user@server` — SSH + systemd, no Docker |

## CLI

| Command | Purpose |
|---------|---------|
| `micro run --prompt "..."` | Generate services + agent, start with interactive console |
| `micro run` | Dev mode: hot reload, gateway, interactive console |
| `micro run -d` | Detached mode (no console) |
| `micro chat` | Standalone chat (when not using micro run) |
| `micro agent list` | List registered agents |
| `micro new myservice` | Scaffold a service |
| `micro call service endpoint '{}'` | Call a service or agent from the CLI |
| `micro build` | Compile production binaries |
| `micro deploy user@server` | Deploy via SSH + systemd |

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

Or use a `micro.mu` config file:

```
service users
    path ./users

service orders
    path ./orders
    depends users
```

## Data Model

Typed persistence with CRUD and queries:

```go
type User struct {
    ID    string `json:"id" model:"key"`
    Name  string `json:"name"`
    Email string `json:"email" model:"index"`
}

db := service.Model()
db.Register(&User{})
db.Create(ctx, &User{ID: "1", Name: "Alice", Email: "alice@example.com"})

var results []*User
db.List(ctx, &results, model.Where("email", "alice@example.com"))
```

Backends: memory (default), SQLite, Postgres.

## AI Providers

Swap providers with a single import — same interface everywhere:

| Provider | Default Model |
|----------|---------------|
| Anthropic | `claude-sonnet-4-20250514` |
| OpenAI | `gpt-4o` |
| Google Gemini | `gemini-2.5-flash` |
| Groq | `llama-3.3-70b-versatile` |
| Mistral | `mistral-large-latest` |
| Together AI | `meta-llama/Llama-3.3-70B-Instruct-Turbo` |
| Atlas Cloud | `deepseek-ai/DeepSeek-V3-0324` |
| Ollama | `llama3.2` (local) |

```go
m := ai.New("anthropic", ai.WithAPIKey(key))
resp, _ := m.Generate(ctx, &ai.Request{Prompt: "hello"})
```

## Examples

- [hello-world](examples/hello-world/) — Basic RPC service
- [multi-service](examples/multi-service/) — Multiple services in one binary
- [mcp](examples/mcp/) — MCP integration with AI agents
- [agent-plan-delegate](examples/agent-plan-delegate/) — Agent planning and multi-agent delegation
- [grpc-interop](examples/grpc-interop/) — Call go-micro from any gRPC client

See [all examples](examples/README.md).

## Docs

- [Getting Started](internal/website/docs/getting-started.md)
- [AI Integration](internal/website/docs/ai-integration.md)
- [Your First Agent](internal/website/docs/guides/your-first-agent.md)
- [0→hero Reference](internal/website/docs/guides/zero-to-hero.md)
- [Agents and Workflows](internal/website/docs/guides/agents-and-workflows.md)
- [Agent Design](internal/docs/AGENT_DESIGN.md)
- [Plan & Delegate](internal/website/docs/guides/plan-delegate.md)
- [Agent Guardrails](internal/website/docs/guides/agent-guardrails.md)
- [Payments (x402)](internal/website/docs/guides/x402-payments.md)
- [MCP & AI Agents](internal/website/docs/mcp.md)
- [Data Model](internal/website/docs/model.md)
- [Deployment](internal/website/docs/deployment.md)
- [Plugins](internal/website/docs/plugins.md)

Package reference: https://pkg.go.dev/go-micro.dev/v6
