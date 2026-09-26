---
title: "Getting Started"
description: "Build an agent, develop services, and use them through conversation."
---

Go Micro is an agent harness and service framework for Go. Start with a
conversation, develop the capabilities you need as services, and let the agent
use their endpoints as tools.

## Start through conversation

Follow the [Quick Start](../quickstart.md) to install Go 1.25+ and the v6 CLI,
configure a provider key, and start `micro chat --provider openai` in a new directory.
With no registered agents, the CLI development agent can generate missing
services, compile and start them, and discover their tools for use in the same
conversation. `micro run --prompt "..." --provider openai` lets you review a design
before generating a project and starting its agent and services.

The generated source stays in your directory. Services started by `micro chat`
stop when that session exits; use `micro run` to work on the generated project
with hot reload. Edit its Go code to change existing services.

## Create your own agent


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
See [Your First Agent](../guides/your-first-agent.md)
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

## Build services for the agent


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

## Add workflows when needed

Services provide capabilities; agents choose which tools to use in response to a
request. Flows coordinate ordered steps or respond to events. With persistent
storage, they can resume from saved step boundaries; interrupted steps may run
again. See [Agents and Workflows](../guides/agents-and-workflows.md) and
[Durability and Recovery](../guides/durability.md) for setup and recovery semantics.

## Development commands

| Command | Purpose |
|---------|---------|
| `micro chat --provider openai` | Use the development agent when none are registered; otherwise route to registered agents |
| `micro run --prompt "..." --provider openai` | Review a design, generate services and an agent, then run them |
| `micro run` | Run the current project with hot reload, gateway, and console |
| `micro run -d` | Run without the console, in the foreground |
| `micro chat assistant --provider openai` | Talk to a specific running agent |
| `micro inspect agent assistant` | Inspect that agent's recorded runs |
| `micro new myservice` | Scaffold a service to implement yourself |
| `micro build` | Compile production binaries |
| `micro deploy user@server` | Deploy via SSH and systemd |

## Examples and troubleshooting

- [Your First Agent](../guides/your-first-agent.md): complete service-and-agent code.
- [Install troubleshooting](../guides/install-troubleshooting.md): toolchain and PATH checks.
- [Debugging your agent](../guides/debugging-agents.md): `micro agent preflight` before running, `micro agent doctor` afterwards, and `micro inspect agent <name>` for recorded runs.
- [No-secret transcript](../guides/no-secret-first-agent.md): use a mock model without an API key. `micro agent demo` prints the command; `micro agent quickcheck` prints troubleshooting steps.
- [Examples index](https://github.com/micro/go-micro/blob/master/examples/INDEX.md): includes the [first-agent](https://github.com/micro/go-micro/tree/master/examples/first-agent) and [support](https://github.com/micro/go-micro/tree/master/examples/support) examples. `micro examples` lists runnable starting points.
- [0→hero reference](../guides/zero-to-hero.md): the optional lifecycle harness, also listed by `micro zero-to-hero`.
- [AI Integration](../ai-integration/index.md): models, service tools, MCP, and agents.
- [Deployment](../deployment.md): build and deploy your application.
