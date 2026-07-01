---
layout: default
---

# Getting Started

<img src="/images/generated/getting-started.jpg" alt="Getting started with Go Micro" style="width: 100%; border-radius: 8px; margin-bottom: 1.5rem;" />

Go Micro has three core abstractions:

| Abstraction | What | Constructor |
|-------------|------|-------------|
| **Service** | Capability — endpoints, data, business logic | `micro.NewService("task")` |
| **Agent** | Intelligence — manages services with an LLM | `micro.NewAgent("task-mgr")` |
| **Flow** | Orchestration — event-driven LLM triggers | `micro.NewFlow("onboard")` |

## Prerequisites

- **Go 1.24+** for development. The `curl` install below gives you the `micro` binary without Go, but `micro run` compiles your services, so you'll want Go installed to build them.
- An **LLM provider key** (Anthropic, OpenAI, Gemini, …) *only* for the AI features — `micro run --prompt`, `micro chat`, and agents. Plain services need no key. Set it before running, e.g. `export ANTHROPIC_API_KEY=sk-ant-...`.

## Install

```bash
# Binary (no Go required)
curl -fsSL https://go-micro.dev/install.sh | sh

# Or with Go
go install go-micro.dev/v6/cmd/micro@latest
```

## Quick Start: Generate from a Prompt

Describe what you need. The AI designs services, writes handlers, compiles, and starts them:

```bash
micro run --prompt "task management system"
```

You'll see the design, confirm, and services + agent start:

```text
Services:
  ● task — Core task management
  ● project — Project organization

Generate? [Y/n]

Micro
  Services:
    ● task
    ● project
  Agents:
    ◆ agent
```

The interactive console lets you talk to your services immediately:

```text
> Create a project called Launch, then add a task called 'Write docs'

→ project_Project_Create({"name":"Launch"})
← {"record":{"id":"p1..."},"success":true}
→ task_Task_Create({"title":"Write docs","project_id":"p1..."})

Created project Launch and added task 'Write docs' to it.
```

The console discovers services from the registry and orchestrates across them via the agent. Use `micro run -d` for detached mode without the console, or `micro chat` as a standalone command.

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

`micro run` gives you:
- **Dashboard** at `http://localhost:8080`
- **API Gateway** at `http://localhost:8080/api/{service}/{method}`
- **Agent Playground** at `http://localhost:8080/agent`
- **MCP Tools** at `http://localhost:8080/mcp/tools`
- **Hot Reload** — auto-rebuild on file changes

`micro new` scaffolds a reflection-based service by default — plain Go types, no code generation, so `go run .` works with nothing else installed. If you prefer Protocol Buffers, add `--proto` (this requires the `protoc` toolchain; the command tells you what to install).

Templates are available for common patterns. These use Protocol Buffers, so they need the `protoc` toolchain (`protoc`, `protoc-gen-go`, `protoc-gen-micro` — `micro new` prints the install commands if they're missing):

```bash
micro new contacts --template crud
micro new events --template pubsub
micro new gateway --template api
```

## Building Agents

For a complete service-backed walkthrough, start with [Your First Agent](guides/your-first-agent.html).

An Agent is an intelligent layer that manages one or more services:

```go
package main

import "go-micro.dev/v6"

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

An agent is a service — it has a proto-defined `Agent.Chat` RPC endpoint and registers in the registry like everything else. It:
- Discovers its services from the registry
- Only sees endpoints from its assigned services (scoped tools)
- Maintains conversation memory in the store (persists across restarts)
- Is callable via `micro call`, the interactive console, or any go-micro client

Use it programmatically:

```go
resp, _ := agent.Ask(ctx, "What tasks are overdue for Alice?")
fmt.Println(resp.Reply)
```

Or via the CLI:

```bash
micro agent list                    # list registered agents
micro call task-mgr Agent.Chat '{"message": "What tasks are overdue?"}'
```

When multiple agents are registered, the console routes to the right agent automatically.

## Event-Driven Flows

A Flow subscribes to a broker topic and triggers an LLM when events arrive. You can define flows in code or run them from the CLI.

**In code:**

```go
f := micro.NewFlow("onboard-user",
    micro.FlowTrigger("events.user.created"),
    micro.FlowPrompt("New user created: {{.Data}}. Send welcome email and create workspace."),
    micro.FlowProvider("anthropic"),
    micro.FlowAPIKey(os.Getenv("MICRO_AI_API_KEY")),
)
f.Register(service.Options().Registry, service.Options().Broker, service.Client())
```

**From the CLI:**

```bash
micro flow run --trigger events.user.created --prompt "New user: {{.Data}}. Send welcome email."
micro flow exec --prompt "Summarize all open tickets and email the report."
```

The flow discovers all services as tools and lets the LLM decide which RPCs to call in response to the event.

## CLI Workflow

| Command | Purpose |
|---------|---------|
| `micro run --prompt "..."` | Generate services + agent, start with interactive console |
| `micro run` | Dev mode: hot reload, gateway, interactive console |
| `micro run -d` | Detached mode (no console) |
| `micro chat` | Standalone chat (when not using micro run) |
| `micro agent list` | List registered agents |
| `micro flow run --trigger <topic>` | Run an event-driven flow |
| `micro flow exec --prompt "..."` | Execute a one-shot flow |
| `micro new myservice` | Scaffold a service |
| `micro call service endpoint '{}'` | Call a service or agent |
| `micro build` | Compile production binaries |
| `micro deploy user@server` | Deploy via SSH + systemd |

## Next Steps

- [AI Integration](ai-integration.html) — how services, agents, MCP, and LLMs fit together
- [Agent Design](https://github.com/micro/go-micro/blob/master/internal/docs/AGENT_DESIGN.md) — the full agent interface specification
- [MCP & AI Agents](mcp.html) — MCP gateway, tool discovery, and auth
- [Data Model](model.html) — typed persistence with CRUD and queries
- [Deployment](deployment.html) — deploy via SSH + systemd
