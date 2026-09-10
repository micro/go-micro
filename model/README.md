# AI Package

The `ai` package provides simple, high-level interfaces for AI model providers. It supports text generation (`Model`), image generation (`ImageModel`), and video generation (`VideoModel`).

## Interfaces

### Text Generation (Model)

The Model interface follows the same patterns as other go-micro packages (Registry, Client, Broker):

```go
type Model interface {
    Init(...Option) error
    Options() Options
    Generate(ctx context.Context, req *Request, opts ...GenerateOption) (*Response, error)
    Stream(ctx context.Context, req *Request, opts ...GenerateOption) (Stream, error)
    String() string
}
```

## Quick Start

```go
import (
    "context"
    "go-micro.dev/v6/model"
    _ "go-micro.dev/v6/model/anthropic"
    _ "go-micro.dev/v6/model/openai"
)

// Create a model
m := model.New("openai",
    model.WithAPIKey("your-api-key"),
    model.WithModel("gpt-4o"),
)

// Generate a response
req := &model.Request{
    Prompt:       "What is Go?",
    SystemPrompt: "You are a helpful programming assistant",
}

resp, err := m.Generate(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Println(resp.Reply)
```

### Image Generation (ImageModel)

```go
type ImageModel interface {
    GenerateImage(ctx context.Context, req *ImageRequest, opts ...GenerateOption) (*ImageResponse, error)
    String() string
}
```

```go
import (
    "go-micro.dev/v6/model"
    _ "go-micro.dev/v6/model/atlascloud"
)

ig := model.NewImage("atlascloud",
    model.WithAPIKey("your-api-key"),
)

resp, err := ig.GenerateImage(context.Background(), &model.ImageRequest{
    Prompt: "A Go gopher in space",
    Size:   "1024x1024",
})

fmt.Println(resp.Images[0].URL)
```

Providers that support image generation: **Atlas Cloud**, **OpenAI**.

### Video Generation (VideoModel)

```go
type VideoModel interface {
    GenerateVideo(ctx context.Context, req *VideoRequest, opts ...GenerateOption) (*VideoResponse, error)
    String() string
}
```

```go
import (
    "go-micro.dev/v6/model"
    _ "go-micro.dev/v6/model/atlascloud"
)

vg := model.NewVideo("atlascloud",
    model.WithAPIKey("your-api-key"),
)

resp, err := vg.GenerateVideo(context.Background(), &model.VideoRequest{
    Prompt:   "Microservices nodes animating with data flowing between them",
    Images:   []string{"https://example.com/diagram.png"}, // optional: image-to-video
    Duration: 6,
})

fmt.Println(resp.URL)
```

Providers that support video generation: **Atlas Cloud**.

## Options

Configure the model using functional options:

```go
m := model.New("anthropic",
    model.WithAPIKey("your-key"),              // Required
    model.WithModel("claude-sonnet-4-20250514"), // Optional, uses provider default
    model.WithBaseURL("https://api.anthropic.com"), // Optional, uses provider default
)
```

You can also update options after creation:

```go
m.Init(
    model.WithModel("gpt-4o-mini"),
    model.WithAPIKey("new-key"),
)
```

## Using Tools

The model can automatically execute tool calls when provided with a tool handler:

```go
// Define a tool handler. It mirrors a go-micro RPC handler: context
// first, the call in, a result out.
toolHandler := func(ctx context.Context, call model.ToolCall) model.ToolResult {
    // Execute the tool and return results
    switch call.Name {
    case "get_weather":
        return model.ToolResult{ID: call.ID, Value: map[string]string{"temp": "72F"}, Content: `{"temp": "72F"}`}
    default:
        return model.ToolResult{ID: call.ID, Content: `{"error": "unknown tool"}`}
    }
}

// Create model with tool handler
m := model.New("openai",
    model.WithAPIKey("your-key"),
    model.WithToolHandler(toolHandler),
)

// Provide tools in the request
req := &model.Request{
    Prompt: "What's the weather?",
    SystemPrompt: "You are a helpful assistant",
    Tools: []model.Tool{
        {
            Name:        "get_weather",
            Description: "Get current weather",
            Properties: map[string]any{
                "location": map[string]any{
                    "type": "string",
                    "description": "City name",
                },
            },
        },
    },
}

// Generate will automatically call tools and return final answer
resp, err := m.Generate(context.Background(), req)
fmt.Println(resp.Answer) // Final answer after tool execution
```

## Response Structure

```go
type Response struct {
    Reply     string      // Initial reply from model
    ToolCalls []ToolCall  // Tools the model wants to call
    Answer    string      // Final answer (after tool execution if handler provided)
}
```

- `Reply`: The model's first response
- `ToolCalls`: List of tools the model requested (if any)
- `Answer`: The final answer after tools are executed (only set if ToolHandler is provided)

## Provider capability matrix

The CLI can print the provider capabilities registered in the current build:

```bash
micro ai providers
```

For automation and docs generation, emit the same matrix as stable JSON:

```bash
micro ai providers --json
```

It reports support from Go Micro's provider registry, so the matrix reflects the model, image, and video interfaces available to this binary rather than external provider marketing claims.

## Supported Providers

### Anthropic Claude

```go
m := model.New("anthropic",
    model.WithAPIKey("sk-ant-..."),
    model.WithModel("claude-sonnet-4-20250514"), // default
)
```

Default model: `claude-sonnet-4-20250514`
Default base URL: `https://api.anthropic.com`

### OpenAI GPT

```go
m := model.New("openai",
    model.WithAPIKey("sk-..."),
    model.WithModel("gpt-4o"), // default
)
```

Default model: `gpt-4o`
Default base URL: `https://api.openmodel.com`

### Google Gemini

```go
m := model.New("gemini",
    model.WithAPIKey("your-key"),
    model.WithModel("gemini-2.5-flash"), // default
)
```

Default model: `gemini-2.5-flash`
Default base URL: `https://generativelanguage.googleapis.com`

Google Gemini uses its own API format with `system_instruction`, `contents` (not `messages`), and `functionDeclarations` for tool calling. The provider handles the translation automatically.

### Groq

```go
m := model.New("groq",
    model.WithAPIKey("your-key"),
    model.WithModel("llama-3.3-70b-versatile"), // default
)
```

Default model: `llama-3.3-70b-versatile`
Default base URL: `https://api.groq.com/openai`

Groq provides ultra-fast inference for open-weight models via an OpenAI-compatible endpoint.

### Mistral

```go
m := model.New("mistral",
    model.WithAPIKey("your-key"),
    model.WithModel("mistral-large-latest"), // default
)
```

Default model: `mistral-large-latest`
Default base URL: `https://api.mistral.ai`

Mistral AI is a European AI company offering high-performance models via an OpenAI-compatible endpoint.

### Together AI

```go
m := model.New("together",
    model.WithAPIKey("your-key"),
    model.WithModel("meta-llama/Llama-3.3-70B-Instruct-Turbo"), // default
)
```

Default model: `meta-llama/Llama-3.3-70B-Instruct-Turbo`
Default base URL: `https://api.together.xyz`

Together AI provides fast inference for open-weight models via an OpenAI-compatible endpoint.

### Atlas Cloud

```go
m := model.New("atlascloud",
    model.WithAPIKey("your-key"),
    model.WithModel("llama-3.3-70b"), // default
)
```

Default model: `llama-3.3-70b`
Default base URL: `https://api.atlascloud.ai`

Atlas Cloud is an enterprise AI infrastructure platform offering high-performance LLM APIs. It exposes an OpenAI-compatible chat completions endpoint with tool calling support.

### MiniMax

```go
m := model.New("minimax",
    model.WithAPIKey("your-key"),
    model.WithModel("MiniMax-M3"), // default
)
```

Default model: `MiniMax-M3`
Default base URL: `https://api.minimax.io`

MiniMax offers its flagship MiniMax-M3 model via an OpenAI-compatible chat completions endpoint.

## Auto-Detection

Use `AutoDetectProvider()` to detect the provider from a base URL:

```go
provider := model.AutoDetectProvider("https://api.anthropic.com")
// Returns "anthropic"

m := model.New(provider, model.WithAPIKey("..."))
```

## Adding a New Provider

See the full **[AI Provider Integration Guide](../internal/website/docs/guides/ai-provider-guide.md)** for a step-by-step walkthrough, checklist, and design notes.

Quick summary:

1. Create `model/yourprovider/yourprovider.go` implementing `model.Model`.
2. Call `model.Register("yourprovider", ...)` in `init()`.
3. Add tests in `model/yourprovider/yourprovider_test.go`.
4. Users enable the provider with a blank import:

```go
import _ "go-micro.dev/v6/model/yourprovider"
```

We welcome contributions and sponsorships from AI infrastructure companies — see the guide for details.

## Comparison with Other Packages

The ai package follows the same patterns as other go-micro packages:

**Registry:**
```go
r := registry.NewRegistry(registry.Addrs("..."))
r.Register(service)
```

**Client:**
```go
c := client.NewClient(client.Retries(3))
c.Call(ctx, req, rsp)
```

**AI:**
```go
m := model.New("openai", model.WithAPIKey("..."))
m.Generate(ctx, req)
```

All use:
- `Init()` to update options
- `Options()` to get current options
- `String()` to get the implementation name
- Functional options pattern

## Testing

```bash
go test ./ai/...
```

## Examples

See the [server implementation](../cmd/micro/server/server.go) for a complete example of using the ai package with tool execution.
