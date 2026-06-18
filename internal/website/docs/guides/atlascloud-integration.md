---
layout: default
title: Atlas Cloud Integration
---

# Atlas Cloud Integration Guide

[Atlas Cloud](https://www.atlascloud.ai/) is an enterprise AI infrastructure platform offering 300+ models across text, image, and video through a unified, OpenAI-compatible API. It is an official Go Micro sponsor and a first-class provider in the `ai` package.

## Quick Start

Install or update Go Micro:

```bash
go get go-micro.dev/v6@latest
```

Import the Atlas Cloud provider and use it:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "go-micro.dev/v6/ai"
    _ "go-micro.dev/v6/ai/atlascloud"
)

func main() {
    m := ai.New("atlascloud",
        ai.WithAPIKey("your-atlas-cloud-key"),
    )

    resp, err := m.Generate(context.Background(), &ai.Request{
        Prompt:       "What is Go Micro?",
        SystemPrompt: "You are a helpful assistant.",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Reply)
}
```

## Configuration

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `ai.WithAPIKey(key)` | *required* | Your Atlas Cloud API key |
| `ai.WithModel(name)` | `llama-3.3-70b` | Model to use (see [Model Selection](#model-selection)) |
| `ai.WithBaseURL(url)` | `https://api.atlascloud.ai` | API base URL |

### Environment Variables

The `micro chat` CLI and `micro run` / `micro server` read configuration from environment variables:

| Variable | Description |
|----------|-------------|
| `ATLASCLOUD_API_KEY` | API key (used by `micro chat --provider atlascloud`) |
| `MICRO_AI_API_KEY` | Generic API key (used by all providers) |
| `MICRO_AI_PROVIDER` | Set to `atlascloud` to select the provider |
| `MICRO_AI_MODEL` | Override the default model |
| `MICRO_AI_BASE_URL` | Override the base URL |

When using `micro chat`, the provider-specific variable takes precedence:

```bash
ATLASCLOUD_API_KEY=your-key micro chat --provider atlascloud
```

When using `micro run` or `micro server`, set the generic variables:

```bash
export MICRO_AI_API_KEY=your-key
export MICRO_AI_BASE_URL=https://api.atlascloud.ai
micro run
```

The server auto-detects Atlas Cloud from the base URL.

## Model Selection

Atlas Cloud offers 300+ models. Some popular choices for the chat completions API:

| Model | Use Case |
|-------|----------|
| `llama-3.3-70b` | General-purpose (default) |
| `deepseek-v4` | Coding and reasoning |
| `qwen-3.6` | Multilingual |

Check [atlascloud.ai](https://www.atlascloud.ai/) for the full model catalog. New SOTA models are available on day zero of release.

```go
m := ai.New("atlascloud",
    ai.WithAPIKey(key),
    ai.WithModel("deepseek-v4"),
)
```

## Image Generation

Atlas Cloud supports text-to-image generation through the `ai.ImageModel` interface. This uses the same OpenAI-compatible `/v1/images/generations` endpoint.

```go
import (
    "context"
    "fmt"

    "go-micro.dev/v6/ai"
    _ "go-micro.dev/v6/ai/atlascloud"
)

func main() {
    ig := ai.NewImage("atlascloud",
        ai.WithAPIKey("your-key"),
    )

    resp, err := ig.GenerateImage(context.Background(), &ai.ImageRequest{
        Prompt: "A Go gopher building microservices, digital art",
        Size:   "1024x1024",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Image returned as URL or base64, depending on the model
    fmt.Println(resp.Images[0].URL)
}
```

### ImageRequest Options

| Field | Default | Description |
|-------|---------|-------------|
| `Prompt` | *required* | Text description of the image |
| `Model` | `gpt-image-1` | Image model to use |
| `Size` | provider default | Image dimensions (e.g. `"1024x1024"`) |
| `N` | `1` | Number of images to generate |

### Available Image Models

Atlas Cloud offers image models including `gpt-image-1`, `flux-2`, `nano-banana-pro`, and more. Check [atlascloud.ai](https://www.atlascloud.ai/) for the full catalog.

```go
ig.GenerateImage(ctx, &ai.ImageRequest{
    Prompt: "A mountain landscape",
    Model:  "flux-2",
    Size:   "1024x1024",
    N:      2,
})
```

The `ai.ImageModel` interface is also implemented by the OpenAI provider, so switching between providers is a one-line change.

## Using with Services (Tool Calling)

Atlas Cloud supports OpenAI-compatible function calling. Combined with Go Micro's `ai.Tools`, your services become tools that the model can call:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "go-micro.dev/v6"
    "go-micro.dev/v6/ai"
    
    _ "go-micro.dev/v6/ai/atlascloud"
)

func main() {
    service := micro.NewService("my-agent")
    service.Init()

    // Discover all services as tools
    tools := ai.NewTools(service.Registry())
    discovered, err := tools.Discover()
    if err != nil {
        log.Fatal(err)
    }

    // Create a model with tool execution
    m := ai.New("atlascloud",
        ai.WithAPIKey("your-key"),
        ai.WithTools(tools),
    )

    // The model can now call your services
    resp, err := m.Generate(context.Background(), &ai.Request{
        Prompt:       "List all users and send each a welcome email",
        SystemPrompt: "You are a service orchestrator.",
        Tools:        discovered,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Answer)
}
```

### How it works

1. `ai.NewTools(registry)` creates a tool set bound to the service registry
2. `tools.Discover()` walks the registry and returns every endpoint as an `ai.Tool`
3. `ai.WithTools(tools)` wires execution into the model — tool calls are routed via RPC
4. When the model decides to call a tool, it routes to the correct service

This works identically across all providers. Swap `"atlascloud"` for `"anthropic"` or `"openai"` and the same services, tools, and handlers work without changes.

## Using with micro chat

`micro chat` is an interactive terminal agent. Start your services, then chat:

```bash
# Terminal 1: start services
micro run

# Terminal 2: chat with Atlas Cloud
ATLASCLOUD_API_KEY=your-key micro chat --provider atlascloud
> what services are running?
> get user alice@example.com
> create a new order for product-42
```

For a single prompt (non-interactive):

```bash
micro chat --provider atlascloud --prompt "list all services"
```

## Using with micro run

The agent playground at `/agent` uses whatever AI provider is configured. To use Atlas Cloud:

```bash
export MICRO_AI_API_KEY=your-atlas-cloud-key
export MICRO_AI_BASE_URL=https://api.atlascloud.ai
micro run
```

Open `http://localhost:8080/agent` and chat with your services through Atlas Cloud.

## Using with MCP

The MCP gateway (`micro mcp serve`) exposes services as tools for external AI agents. Atlas Cloud's models can be used by any MCP-compatible agent that connects to the gateway. The gateway itself doesn't depend on a specific AI provider — it serves tools over MCP, and the agent on the other end chooses which model to use.

## Swapping Providers

All Go Micro AI providers implement the same `ai.Model` interface. To switch from Atlas Cloud to another provider, change the import and the provider name:

```go
// Atlas Cloud
import _ "go-micro.dev/v6/ai/atlascloud"
m := ai.New("atlascloud", ai.WithAPIKey(key))

// Anthropic
import _ "go-micro.dev/v6/ai/anthropic"
m := ai.New("anthropic", ai.WithAPIKey(key))

// OpenAI
import _ "go-micro.dev/v6/ai/openai"
m := ai.New("openai", ai.WithAPIKey(key))
```

The rest of your code — tool discovery, handler wiring, request/response handling — stays the same.

## API Compatibility

Atlas Cloud exposes an OpenAI-compatible `/v1/chat/completions` endpoint. This means:

- **Existing OpenAI SDK code** works by changing the base URL
- **Tool calling** uses the same `tools` and `tool_calls` format as OpenAI
- **Streaming** follows the OpenAI SSE format (when implemented)

If you're already using the `openai` provider, you can point it at Atlas Cloud directly:

```go
import _ "go-micro.dev/v6/ai/openai"

m := ai.New("openai",
    ai.WithAPIKey("your-atlas-cloud-key"),
    ai.WithBaseURL("https://api.atlascloud.ai"),
    ai.WithModel("llama-3.3-70b"),
)
```

The dedicated `atlascloud` provider simply sets these defaults for you.

## Links

- [Atlas Cloud](https://www.atlascloud.ai/) — Sign up and get an API key
- [AI Provider Integration Guide](/docs/guides/ai-provider-guide) — How providers are built
- [ai.Tools](https://pkg.go.dev/go-micro.dev/v6/ai.Tools) — Service-to-tool discovery
- [Blog: Atlas Cloud Sponsors Go Micro](/blog/8) — Announcement post
