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
- **No LLM provider key is required** for the first run below. Add an Anthropic, OpenAI, Gemini, or other provider key only when you reach the provider-backed generation and chat steps.

## Install

```bash
# Binary (no Go required)
curl -fsSL https://go-micro.dev/install.sh | sh

# Or with Go
go install go-micro.dev/v6/cmd/micro@latest
```

## Quick Start: Scaffold, Run, Call

Start with the path that proves the runtime works before any provider setup: install the CLI, scaffold one service, run it locally, then call it through the gateway.

```bash
micro new helloworld
cd helloworld
micro run
```

In another terminal, call the generated service:

```bash
curl -X POST http://localhost:8080/api/helloworld/Helloworld.Call \
  -H 'Content-Type: application/json' -d '{"name":"World"}'
```

That install → scaffold → run → call loop is the 0→1 contract. It requires Go and the `micro` binary, but no LLM key. Once this succeeds, you know the local runtime, hot reload, gateway, and service registration are working.


### First-agent on-ramp

After this quick start, follow the agent path in order:

1. `micro agent demo` — print the provider-free first-agent demo command and next docs steps from the installed CLI.
2. [Smallest first-agent example](https://github.com/micro/go-micro/tree/master/examples/first-agent) — run one service-backed agent with a mock model and no provider key.
3. [No-secret first-agent transcript](guides/no-secret-first-agent.html) — run a useful support agent with a mock model before setting up a provider key.
4. [Your First Agent](guides/your-first-agent.html) — build a service-backed agent and talk to it with `micro chat`.
5. [Debugging your agent](guides/debugging-agents.html) — inspect service registration, tool calls, run history, memory, provider failures, and flow handoffs when the agent surprises you.
6. [0→hero reference path](guides/zero-to-hero.html) — prove the full scaffold → run → chat → inspect → deploy dry-run lifecycle with commands exercised by `make harness`.

## Write a Service

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


## Generate from a Prompt — with an LLM key

After the no-secret path works, set a provider key if you want Go Micro to design services and an agent from a prompt:

```bash
export ANTHROPIC_API_KEY=sk-ant-...   # or OPENAI_API_KEY, GEMINI_API_KEY, ...
micro run --prompt "task management system" --provider anthropic
```

You'll see the design, confirm it, and then services plus an agent start:

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

Use the interactive console, `micro run -d` plus `micro chat`, or the agent playground to talk to the generated services.

Before your first provider-backed agent run, check the local path with:

```bash
micro agent preflight
```

The preflight is read-only: it verifies Go 1.24+, the `micro` binary, provider-key setup, and whether the default `micro run` gateway port is free, without calling an LLM provider. When a check fails it prints the exact fix plus the next guide to open, so the scaffold → run → chat path stays walkable.

## Building Agents

For a complete service-backed walkthrough, start with [Your First Agent](guides/your-first-agent.html). If you want to run before you write, use [`examples/support`](https://github.com/micro/go-micro/tree/master/examples/support) for the full services → agents → workflows lifecycle or [`examples/agent-plan-delegate`](https://github.com/micro/go-micro/tree/master/examples/agent-plan-delegate) for the smallest multi-agent planning/delegation path.

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

- [Learn by Example](examples/) — runnable examples mapped to services, agents, and workflows
- [0→hero Reference](guides/zero-to-hero.html) — the maintained no-secret lifecycle contract
- [AI Integration](ai-integration.html) — how services, agents, MCP, and LLMs fit together
- [Agent Design](https://github.com/micro/go-micro/blob/master/internal/docs/AGENT_DESIGN.md) — the full agent interface specification
- [MCP & AI Agents](mcp.html) — MCP gateway, tool discovery, and auth
- [Data Model](model.html) — typed persistence with CRUD and queries
- [Deployment](deployment.html) — deploy via SSH + systemd
