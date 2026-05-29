---
layout: default
title: AI Integration
---

# AI Integration

Go Micro is an AI-native microservices framework. Every service you build is automatically accessible to AI agents, and every service can call AI models. This page explains how the pieces fit together.

<img src="/images/generated/mcp-agent.png" alt="AI integration architecture" style="width: 100%; border-radius: 8px; margin: 1rem 0 1.5rem;" />

## The Stack

```
Your Services          →  write Go handlers, register with the framework
    ↓
Registry               →  automatic service discovery (mDNS, Consul, etcd)
    ↓
MCP Gateway            →  exposes every endpoint as an AI-callable tool
    ↓
ai/tools               →  discovers services + executes RPCs programmatically
    ↓
ai.Model               →  calls LLMs (Anthropic, OpenAI, Gemini, Atlas Cloud, ...)
    ↓
micro chat / your app  →  orchestrates services through natural language
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
service := micro.New("users")
service.Handle(handler.New())
service.Run() // registers with the registry
```

Pluggable: mDNS (default, zero config), Consul, etcd, NATS.

### 3. MCP Gateway (services → tools)

The MCP gateway walks the registry and exposes every endpoint as a tool via the [Model Context Protocol](https://modelcontextprotocol.io/):

```go
// One line to expose all services as AI tools
service := micro.New("myservice", mcp.WithMCP(":3001"))
```

Or run it standalone:

```bash
micro mcp serve              # stdio for Claude Code
micro mcp serve --address :3000  # HTTP for web agents
```

Any MCP-compatible agent (Claude Code, ChatGPT, custom agents) can discover and call your services.

### 4. ai/tools (discover + execute)

The `ai/tools` package extracts the MCP gateway's logic into a reusable building block:

```go
import "go-micro.dev/v5/ai/tools"

set := tools.New(service.Registry())
discovered, _ := set.Discover()    // []ai.Tool from all registered services
handler := set.Handler(service.Client()) // executes tool calls via RPC
```

This is what powers `micro chat` and the agent playground. You can use it directly in your own services to build agentic workflows.

### 5. ai.Model (LLM providers)

The `ai` package provides a pluggable interface for calling LLMs:

```go
import (
    "go-micro.dev/v5/ai"
    _ "go-micro.dev/v5/ai/anthropic"
)

m := ai.New("anthropic", ai.WithAPIKey(key))
resp, _ := m.Generate(ctx, &ai.Request{
    Prompt: "What users are in the system?",
    Tools:  discovered,  // from ai/tools
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

### 7. Your app (programmatic use)

Use the same building blocks in your own services:

```go
// Subscribe to events and let an LLM decide what to do
broker.Subscribe("user.created", func(e broker.Event) error {
    prompt := fmt.Sprintf("New user: %s. Send welcome email and create workspace.", 
        string(e.Message().Body))
    resp, _ := hist.Generate(ctx, m, prompt, discovered)
    log.Info(resp.Answer)
    return nil
})
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
