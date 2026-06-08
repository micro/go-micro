# Go Micro [![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/go-micro.dev/v5?tab=doc) [![Go Report Card](https://goreportcard.com/badge/github.com/go-micro/go-micro)](https://goreportcard.com/report/github.com/go-micro/go-micro)

Go Micro is a framework for building services and agents in Go.

Write services — they register, discover each other, and communicate via RPC and events. Every endpoint is automatically an AI-callable tool via [MCP](https://modelcontextprotocol.io/). Build agents to manage them intelligently. Both are Go code, both use the same primitives, both deploy the same way.

## Sponsors

<a href="https://go-micro.dev/blog/3"><img src="https://upload.wikimedia.org/wikipedia/commons/7/78/Anthropic_logo.svg" height="26" /></a>
&nbsp;&nbsp;
<a href="https://go-micro.dev/blog/8"><img src="https://www.atlascloud.ai/logo.svg" height="26" /></a>

## Quick Start

Install the CLI:

```bash
# Binary (no Go required)
curl -fsSL https://go-micro.dev/install.sh | sh

# Or with Go
go install go-micro.dev/v5/cmd/micro@v5.27.0
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

Created Work category and added 'Finish report' task to it.
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

## Writing Services

Under the hood, a service is a struct with methods. Doc comments and `@example` tags become tool descriptions for AI agents automatically.

```go
package main

import (
    "go-micro.dev/v5"
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
    service := micro.New("greeter")
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

Every agent gets two built-in capabilities, exposed as tools — no extra setup, no harness:

- **`plan`** — for multi-step work, the agent records an ordered plan in its store-backed memory and stays oriented across turns.
- **`delegate`** — the agent hands a self-contained subtask to another agent. If a registered agent already owns the relevant services, the hand-off goes over RPC to that agent; otherwise a focused, short-lived sub-agent is created for the subtask with its own isolated context.

This keeps intelligence distributed: an agent doesn't need to know *how* to do everything, only *who* does. See [examples/agent-plan-delegate](examples/agent-plan-delegate/).

```go
// A sub-agent is just an agent — created with New, talked to with Ask.
// delegate-first: reuse a registered agent, or spin up a focused one.
resp, _ := agent.Ask(ctx, "Plan the launch, create the tasks, and have comms notify the owner.")
```

## Features

### AI

| Feature | Details |
|---------|---------|
| Agents | `micro.NewAgent()` — intelligent layer that manages services |
| Plan & delegate | Built-in agent tools — plan multi-step work, delegate subtasks to other agents |
| Guardrails | `MaxSteps` (stopping condition) and `ApproveTool` (human-in-the-loop) on every agent |
| Workflows | `micro.NewFlow()` — event-driven; runs a step or triggers an agent |
| MCP gateway | Every endpoint is an AI tool automatically |
| 7 LLM providers | Anthropic, OpenAI, Gemini, Groq, Mistral, Together, Atlas Cloud |
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
users := micro.New("users", micro.Address(":9001"))
orders := micro.New("orders", micro.Address(":9002"))

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
| Together AI | `Llama-3.3-70B-Instruct-Turbo` |
| Atlas Cloud | `llama-3.3-70b` |

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
- [Agents and Workflows](internal/website/docs/guides/agents-and-workflows.md)
- [Agent Design](internal/docs/AGENT_DESIGN.md)
- [Plan & Delegate](internal/website/docs/guides/plan-delegate.md)
- [MCP & AI Agents](internal/website/docs/mcp.md)
- [Data Model](internal/website/docs/model.md)
- [Deployment](internal/website/docs/deployment.md)
- [Plugins](internal/website/docs/plugins.md)

Package reference: https://pkg.go.dev/go-micro.dev/v5
