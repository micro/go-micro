---
layout: default
title: AI Integration
---

# AI Integration

Go Micro is an AI-native microservices framework. Every service you build is automatically accessible to AI agents, and every service can call AI models. This page explains how the pieces fit together.

<img src="/images/generated/mcp-agent.jpg" alt="AI integration architecture" style="width: 100%; border-radius: 8px; margin: 1rem 0 1.5rem;" />

## The Stack

```
Your Services           →  write Go handlers, register with the framework
    ↓
Registry                →  automatic service discovery (mDNS, Consul, etcd)
    ↓
Gateways                →  micro api (HTTP→RPC) / micro mcp (MCP tools)
    ↓
ai.Tools                →  discovers services + executes RPCs programmatically
    ↓
ai.Model                →  calls LLMs (Anthropic, OpenAI, Gemini, Atlas Cloud, ...)
    ↓
agent / flow / micro chat  →  agent-managed, event-driven, or interactive orchestration
```

Every layer is optional. You can use go-micro without AI. You can use the `ai` package without MCP. But when you stack them, you get services that AI agents can discover and orchestrate automatically.

## Layer by Layer

### 1. Services (your code)

Write normal Go handlers. Add doc comments for AI tool descriptions:

```go
// CreateUser creates a new user account.
// @example {"name": "Alice", "email": "alice@example.com"}
func (h *Users) CreateUser(ctx context.Context, req *pb.CreateRequest, rsp *pb.CreateResponse) error {
    // your business logic
}
```

The doc comment becomes the tool description. The `@example` tag gives the LLM a usage hint. No AI-specific code in your handler.

### 2. Registry (service discovery)

Services register automatically. The registry is the source of truth for what's running:

```go
service := micro.NewService("users")
service.Handle(handler.New())
service.Run() // registers with the registry
```

Pluggable: mDNS (default, zero config), Consul, etcd, NATS.

### 3. MCP Gateway (services → tools)

The MCP gateway walks the registry and exposes every endpoint as a tool via the [Model Context Protocol](https://modelcontextprotocol.io/):

```go
// One line to expose all services as AI tools
service := micro.NewService("myservice", mcp.WithMCP(":3001"))
```

Or run it standalone:

```bash
micro mcp serve              # stdio for Claude Code
micro mcp serve --address :3000  # HTTP for web agents
```

Any MCP-compatible agent (Claude Code, ChatGPT, custom agents) can discover and call your services.

### 4. ai.Tools (discover + execute)

`ai.Tools` turns registered services into LLM-callable tools — discovery plus RPC execution in one type:

```go
tools := ai.NewTools(service.Registry())
discovered, _ := tools.Discover()  // []ai.Tool from all registered services

// Wire execution into a model with one option:
m := ai.New("anthropic", ai.WithAPIKey(key), ai.WithTools(tools))
```

This is what powers `micro chat` and the agent playground. You can use it directly in your own services to build agentic workflows.

### 5. ai.Model (LLM providers)

The `ai` package provides a pluggable interface for calling LLMs:

```go
import (
    "go-micro.dev/v6/ai"
    _ "go-micro.dev/v6/ai/anthropic"
)

m := ai.New("anthropic", ai.WithAPIKey(key))
resp, _ := m.Generate(ctx, &ai.Request{
    Prompt: "What users are in the system?",
    Tools:  discovered,  // from ai.Tools
})
```

Seven text providers, two image providers, one video provider. Same interface, swap with an import.

| Provider | Text | Image | Video |
|----------|------|-------|-------|
| Anthropic | yes | | |
| OpenAI | yes | yes | |
| Google Gemini | yes | | |
| Atlas Cloud | yes | yes | yes |
| Groq | yes | | |
| Mistral | yes | | |
| Together AI | yes | | |

### 6. micro chat (orchestration)

The CLI ties it all together — discovers services, builds the tool list, and lets you talk to your services:

```bash
ANTHROPIC_API_KEY=sk-ant-... micro chat --provider anthropic
> list all users
> send a welcome email to alice@example.com
> create an order for product-42
```

Multi-turn conversation with `ai.History` — the model remembers context across turns. Type `reset` to clear history.

### 7. micro flow (event-driven orchestration)

Subscribe to broker events and let an LLM orchestrate the response:

```go
import "go-micro.dev/v6/flow"

f := flow.New("onboard",
    flow.Trigger("events.user.created"),
    flow.Prompt("New user: {{.Data}}. Send welcome email and create workspace."),
    flow.Provider("anthropic"),
    flow.APIKey(key),
)
f.Register(service.Registry(), service.Options().Broker, service.Client())
```

Or from the CLI:

```bash
micro flow run --trigger events.user.created \
  --prompt "New user: {{.Data}}. Send welcome email." \
  --provider anthropic

micro flow exec --prompt "List all users" --provider anthropic
```

### 8. micro api (HTTP gateway)

A standalone HTTP-to-RPC gateway for exposing services over HTTP without the full dashboard:

```bash
micro api                    # listen on :8080
micro api --address :3000    # custom port

# Call services through the gateway
curl -XPOST -d '{"name":"Alice"}' http://localhost:8080/greeter/Greeter.Hello
```

## What You Don't Need

- **No agent framework** — the building blocks compose; you don't need a LangChain or CrewAI equivalent
- **No special handler code** — your services are normal Go handlers with doc comments
- **No API key to use MCP** — external agents bring their own models; your services just expose tools
- **No vendor lock-in** — every provider implements the same interface; swap with one import

## Getting Started

The fastest path:

```bash
# Create a service with MCP enabled
micro new myservice --template crud
cd myservice

# Run it
micro run

# Chat with it
ANTHROPIC_API_KEY=sk-ant-... micro chat --provider anthropic
> list all records
```

See also:
- [MCP Documentation](/docs/mcp.html) — detailed MCP gateway guide
- [Atlas Cloud Integration](/docs/guides/atlascloud-integration.html) — using Atlas Cloud as a provider
- [AI Provider Guide](/docs/guides/ai-provider-guide.html) — adding new providers
- [gRPC Interop Example](https://github.com/micro/go-micro/tree/master/examples/grpc-interop) — calling go-micro from standard gRPC clients
