# Go Micro [![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/go-micro.dev/v5?tab=doc) [![Go Report Card](https://goreportcard.com/badge/github.com/go-micro/go-micro)](https://goreportcard.com/report/github.com/go-micro/go-micro)

Go Micro is a framework for building microservices that AI agents can use.

Write services in Go. They register, discover each other, and communicate via RPC and events. Every endpoint is automatically an AI-callable tool via [MCP](https://modelcontextprotocol.io/). An agent orchestrates across services so they don't have to call each other.

## Quick Start

Generate services from a description, start them, and talk to them:

```bash
go install go-micro.dev/v5/cmd/micro@latest

micro run --prompt "a task management system with categories" --provider anthropic
micro chat --provider anthropic
> Create a Work category, then add a task called 'Finish report' to it
```

Or scaffold a single service by hand:

```bash
micro new helloworld
cd helloworld
micro run
```

Open http://localhost:8080 to see the dashboard, call endpoints, and chat with your services.

## Sponsors

<a href="https://go-micro.dev/blog/3"><img src="https://upload.wikimedia.org/wikipedia/commons/7/78/Anthropic_logo.svg" height="26" /></a>
&nbsp;&nbsp;
<a href="https://go-micro.dev/blog/8"><img src="https://www.atlascloud.ai/logo.svg" height="26" /></a>

## How It Works

### 1. Write a Service

A service is a struct with methods. Doc comments and `@example` tags become tool descriptions for AI agents.

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

### 2. Run It

`micro run` starts your service with an API gateway, agent playground, and hot reload:

```bash
micro run
# Dashboard:   http://localhost:8080
# API:         http://localhost:8080/api/{service}/{method}
# Agent:       http://localhost:8080/agent
# MCP Tools:   http://localhost:8080/mcp/tools
```

### 3. Talk to It

`micro chat` discovers services from the registry, exposes every endpoint as a tool, and lets an LLM orchestrate:

```bash
micro chat --provider anthropic
> Say hello to Alice
→ greeter_Say_Hello({"name":"Alice"})
← {"message":"Hello Alice"}
```

When you need a capability that doesn't exist, the agent generates a new service mid-conversation, compiles it, starts it, and uses it immediately. [Read more](https://go-micro.dev/blog/13).

## Features

| Category | What | Details |
|----------|------|---------|
| **Discovery** | Service registry | mDNS (default), Consul, etcd |
| **Communication** | RPC client/server | gRPC transport, load balancing, streaming |
| **Messaging** | Pub/sub events | NATS, RabbitMQ, HTTP broker |
| **Storage** | Key-value store | File (bbolt), Postgres, NATS KV |
| **Data** | Typed model layer | CRUD + queries, SQLite/Postgres backends |
| **AI** | MCP gateway | Every endpoint is an AI tool automatically |
| **AI** | 7 LLM providers | Anthropic, OpenAI, Gemini, Groq, Mistral, Together, Atlas Cloud |
| **AI** | Agent orchestration | `micro chat` — LLM calls services as tools |
| **AI** | Service generation | `micro run --prompt` — describe a system, get running services |
| **DX** | Hot reload | `micro run` watches files, rebuilds on change |
| **DX** | Templates | `micro new --template crud/pubsub/api` |
| **Deploy** | One-command deploy | `micro deploy user@server` — SSH + systemd, no Docker |
| **Plugins** | Everything swappable | All abstractions are Go interfaces |

## CLI Workflow

| Command | Purpose |
|---------|---------|
| `micro new myservice` | Scaffold a service |
| `micro run` | Dev mode: hot reload, gateway, agent playground |
| `micro run --prompt "..."` | Generate services from a description and run them |
| `micro call service endpoint '{"key":"val"}'` | Call a service from the CLI |
| `micro chat --provider anthropic` | Talk to services through an LLM |
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
- [MCP & AI Agents](internal/website/docs/mcp.md)
- [Data Model](internal/website/docs/model.md)
- [Deployment](internal/website/docs/deployment.md)
- [Plugins](internal/website/docs/plugins.md)

Package reference: https://pkg.go.dev/go-micro.dev/v5

## Adopters

- [Sourse](https://sourse.eu) — Earth observation platform with embedded Kubernetes and SaaS built on Go Micro.
