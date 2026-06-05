# Go Micro [![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/go-micro.dev/v5?tab=doc) [![Go Report Card](https://goreportcard.com/badge/github.com/go-micro/go-micro)](https://goreportcard.com/report/github.com/go-micro/go-micro)

Go Micro is a framework for building microservices that AI agents can use.

Write services in Go. They register, discover each other, and communicate via RPC and events. Every endpoint is automatically an AI-callable tool via [MCP](https://modelcontextprotocol.io/). An agent orchestrates across services so they don't have to call each other.

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
go install go-micro.dev/v5/cmd/micro@v5.25.0
```

Generate services from a description and start them:

```bash
micro run --prompt "a task management system with categories" --provider anthropic
```

The AI designs the architecture, you review it, then it generates handlers with real business logic, compiles them, and starts them:

```
Services:
  ● category — Manages task categories
  ● task — Task management with status tracking

Generate? [Y/n]
```

```
Micro

  Dashboard   http://localhost:8080
  API         http://localhost:8080/api/{service}/{method}
  Agent       http://localhost:8080/agent

  Services:
    ● category
    ● task
```

Talk to your services through an agent:

```bash
micro chat --provider anthropic
> Create a Work category, then add a task called 'Finish report' to it
```

The agent discovers services from the registry, sees every endpoint as a tool, and orchestrates across them:

```
→ category_Category_Create({"name":"Work","user_id":"user1"})
← {"record":{"id":"f633...","name":"Work"},"success":true}
→ task_Task_Create({"title":"Finish report","category_id":"f633..."})
← {"record":{"id":"a1b2...","title":"Finish report","status":"pending"}}

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

An Agent is the intelligence layer that manages services. It's a first-class abstraction alongside Service:

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
resp, _ := agent.Chat(ctx, "What tasks are overdue?")
fmt.Println(resp.Reply)
```

Multiple agents coordinate through the broker — each manages its domain, `micro chat` routes to the right one.

```bash
micro agent list                    # list registered agents
micro agent chat task-mgr           # talk to a specific agent
```

## Features

| Category | What | Details |
|----------|------|---------|
| **AI** | Agents | `micro.NewAgent()` — intelligent layer that manages services |
| **AI** | Flows | `micro.NewFlow()` — event-driven LLM orchestration |
| **AI** | MCP gateway | Every endpoint is an AI tool automatically |
| **AI** | 7 LLM providers | Anthropic, OpenAI, Gemini, Groq, Mistral, Together, Atlas Cloud |
| **AI** | Chat router | `micro chat` routes to agents or calls services directly |
| **AI** | Service generation | `micro run --prompt` — describe a system, get running services |
| **Discovery** | Service registry | mDNS (default), Consul, etcd |
| **Communication** | RPC client/server | gRPC transport, load balancing, streaming |
| **Messaging** | Pub/sub events | NATS, RabbitMQ, HTTP broker |
| **Storage** | Key-value store | File (bbolt), Postgres, NATS KV |
| **Data** | Typed model layer | CRUD + queries, SQLite/Postgres backends |
| **DX** | Hot reload | `micro run` watches files, rebuilds on change |
| **DX** | Templates | `micro new --template crud/pubsub/api` |
| **Deploy** | One-command deploy | `micro deploy user@server` — SSH + systemd, no Docker |
| **Plugins** | Everything swappable | All abstractions are Go interfaces |

## CLI

| Command | Purpose |
|---------|---------|
| `micro run --prompt "..."` | Generate services from a description and run them |
| `micro chat` | Route messages to agents or call services directly |
| `micro agent list` | List registered agents |
| `micro agent describe <name>` | Show agent details |
| `micro new myservice` | Scaffold a service |
| `micro run` | Dev mode: hot reload, gateway, agent playground |
| `micro call service endpoint '{}'` | Call a service from the CLI |
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
- [grpc-interop](examples/grpc-interop/) — Call go-micro from any gRPC client

See [all examples](examples/README.md).

## Docs

- [Getting Started](internal/website/docs/getting-started.md)
- [AI Integration](internal/website/docs/ai-integration.md)
- [Agent Design](internal/docs/AGENT_DESIGN.md)
- [MCP & AI Agents](internal/website/docs/mcp.md)
- [Data Model](internal/website/docs/model.md)
- [Deployment](internal/website/docs/deployment.md)
- [Plugins](internal/website/docs/plugins.md)

Package reference: https://pkg.go.dev/go-micro.dev/v5

## Adopters

- [Sourse](https://sourse.eu) — Earth observation platform with embedded Kubernetes and SaaS built on Go Micro.
