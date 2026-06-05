---
layout: default
---

# Getting Started

<img src="/images/generated/getting-started.png" alt="Getting started with Go Micro" style="width: 100%; border-radius: 8px; margin-bottom: 1.5rem;" />

Go Micro has three core abstractions:

| Abstraction | What | Constructor |
|-------------|------|-------------|
| **Service** | Capability — endpoints, data, business logic | `micro.New("task")` |
| **Agent** | Intelligence — manages services with an LLM | `micro.NewAgent("task-mgr")` |
| **Flow** | Orchestration — event-driven LLM triggers | `micro.NewFlow("onboard")` |

## Install

```bash
# Binary (no Go required)
curl -fsSL https://go-micro.dev/install.sh | sh

# Or with Go
go install go-micro.dev/v5/cmd/micro@v5.25.0
```

## Quick Start: Generate from a Prompt

Describe what you need. The AI designs services, writes handlers, compiles, and starts them:

```bash
micro run --prompt "task management system"
```

You'll see the design, confirm, and services start:

```text
Services:
  ● task — Core task management
  ● project — Project organization

Generate? [Y/n]

Micro
  Dashboard   http://localhost:8080
  Services:
    ● task
    ● project
```

Talk to your services through an agent:

```bash
micro chat --provider anthropic
> Create a project called Launch, then add a task called 'Write docs'
```

The agent discovers services, calls the right endpoints, and orchestrates across them.

## Quick Start: Write a Service

Create and run a service manually:

```bash
micro new helloworld
cd helloworld
micro run
```

Open http://localhost:8080 to see the dashboard, call endpoints, and chat with your service.

A service is a Go struct with methods. Doc comments and `@example` tags become tool descriptions for AI agents:

```go
package main

import "go-micro.dev/v5"

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

`micro run` gives you:
- **Dashboard** at `http://localhost:8080`
- **API Gateway** at `http://localhost:8080/api/{service}/{method}`
- **Agent Playground** at `http://localhost:8080/agent`
- **MCP Tools** at `http://localhost:8080/mcp/tools`
- **Hot Reload** — auto-rebuild on file changes

Templates are available for common patterns:

```bash
micro new contacts --template crud
micro new events --template pubsub
micro new gateway --template api
```

## Building Agents

An Agent is an intelligent layer that manages one or more services:

```go
package main

import "go-micro.dev/v5"

func main() {
    agent := micro.NewAgent("task-mgr",
        micro.AgentServices("task", "project"),
        micro.AgentPrompt("You manage tasks and projects. You understand deadlines, priorities, and assignments."),
        micro.AgentProvider("anthropic"),
        micro.AgentAPIKey("sk-ant-..."),
    )
    agent.Run()
}
```

The agent:
- Discovers its services from the registry
- Only sees endpoints from its assigned services (scoped tools)
- Maintains conversation memory in the store (persists across restarts)
- Registers itself so `micro chat` can route to it

Use it programmatically:

```go
resp, _ := agent.Chat(ctx, "What tasks are overdue for Alice?")
fmt.Println(resp.Reply)
```

Or via the CLI:

```bash
micro agent list                    # list registered agents
micro agent describe task-mgr       # show agent details
```

When multiple agents are registered, `micro chat` becomes a router — it classifies intent and dispatches to the right agent automatically.

## Event-Driven Flows

A Flow subscribes to a broker topic and triggers an LLM when events arrive:

```go
f := micro.NewFlow("onboard-user",
    micro.FlowTrigger("events.user.created"),
    micro.FlowPrompt("New user created: {{.Data}}. Send welcome email and create workspace."),
    micro.FlowProvider("anthropic"),
    micro.FlowAPIKey("sk-ant-..."),
)
f.Register(service.Options().Registry, service.Options().Broker, service.Client())
```

The flow discovers all services as tools and lets the LLM decide which RPCs to call in response to the event.

## CLI Workflow

| Command | Purpose |
|---------|---------|
| `micro run --prompt "..."` | Generate services from a description and run them |
| `micro chat` | Route messages to agents or call services directly |
| `micro agent list` | List registered agents |
| `micro new myservice` | Scaffold a service |
| `micro run` | Dev mode: hot reload, gateway, agent playground |
| `micro call service endpoint '{}'` | Call a service from the CLI |
| `micro build` | Compile production binaries |
| `micro deploy user@server` | Deploy via SSH + systemd |

## Next Steps

- [AI Integration](ai-integration.html) — how services, agents, MCP, and LLMs fit together
- [Agent Design](https://github.com/micro/go-micro/blob/master/internal/docs/AGENT_DESIGN.md) — the full agent interface specification
- [MCP & AI Agents](mcp.html) — MCP gateway, tool discovery, and auth
- [Data Model](model.html) — typed persistence with CRUD and queries
- [Deployment](deployment.html) — deploy via SSH + systemd
