# AI Package

The `ai` package provides a simple, high-level interface for AI model providers like Anthropic Claude and OpenAI GPT.

## Interface

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
    "go-micro.dev/v5/ai"
    _ "go-micro.dev/v5/ai/anthropic"
    _ "go-micro.dev/v5/ai/openai"
)

// Create a model
m := ai.New("openai",
    ai.WithAPIKey("your-api-key"),
    ai.WithModel("gpt-4o"),
)

// Generate a response
req := &ai.Request{
    Prompt:       "What is Go?",
    SystemPrompt: "You are a helpful programming assistant",
}

resp, err := m.Generate(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Println(resp.Reply)
```

## Options

Configure the model using functional options:

```go
m := ai.New("anthropic",
    ai.WithAPIKey("your-key"),              // Required
    ai.WithModel("claude-sonnet-4-20250514"), // Optional, uses provider default
    ai.WithBaseURL("https://api.anthropic.com"), // Optional, uses provider default
)
```

You can also update options after creation:

```go
m.Init(
    ai.WithModel("gpt-4o-mini"),
    ai.WithAPIKey("new-key"),
)
```

## Using Tools

The model can automatically execute tool calls when provided with a tool handler:

```go
// Define a tool handler
toolHandler := func(name string, input map[string]any) (result any, content string) {
    // Execute the tool and return results
    switch name {
    case "get_weather":
        return map[string]string{"temp": "72F"}, `{"temp": "72F"}`
    default:
        return nil, `{"error": "unknown tool"}`
    }
}

// Create model with tool handler
m := ai.New("openai",
    ai.WithAPIKey("your-key"),
    ai.WithToolHandler(toolHandler),
)

// Provide tools in the request
req := &ai.Request{
    Prompt: "What's the weather?",
    SystemPrompt: "You are a helpful assistant",
    Tools: []ai.Tool{
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

## Supported Providers

### Anthropic Claude

```go
m := ai.New("anthropic",
    ai.WithAPIKey("sk-ant-..."),
    ai.WithModel("claude-sonnet-4-20250514"), // default
)
```

Default model: `claude-sonnet-4-20250514`
Default base URL: `https://api.anthropic.com`

### OpenAI GPT

```go
m := ai.New("openai",
    ai.WithAPIKey("sk-..."),
    ai.WithModel("gpt-4o"), // default
)
```

Default model: `gpt-4o`
Default base URL: `https://api.openai.com`

### Google Gemini

```go
m := ai.New("gemini",
    ai.WithAPIKey("your-key"),
    ai.WithModel("gemini-2.5-flash"), // default
)
```

Default model: `gemini-2.5-flash`
Default base URL: `https://generativelanguage.googleapis.com`

Google Gemini uses its own API format with `system_instruction`, `contents` (not `messages`), and `functionDeclarations` for tool calling. The provider handles the translation automatically.

### Groq

```go
m := ai.New("groq",
    ai.WithAPIKey("your-key"),
    ai.WithModel("llama-3.3-70b-versatile"), // default
)
```

Default model: `llama-3.3-70b-versatile`
Default base URL: `https://api.groq.com/openai`

Groq provides ultra-fast inference for open-weight models via an OpenAI-compatible endpoint.

### Mistral

```go
m := ai.New("mistral",
    ai.WithAPIKey("your-key"),
    ai.WithModel("mistral-large-latest"), // default
)
```

Default model: `mistral-large-latest`
Default base URL: `https://api.mistral.ai`

Mistral AI is a European AI company offering high-performance models via an OpenAI-compatible endpoint.

### Together AI

```go
m := ai.New("together",
    ai.WithAPIKey("your-key"),
    ai.WithModel("meta-llama/Llama-3.3-70B-Instruct-Turbo"), // default
)
```

Default model: `meta-llama/Llama-3.3-70B-Instruct-Turbo`
Default base URL: `https://api.together.xyz`

Together AI provides fast inference for open-weight models via an OpenAI-compatible endpoint.

### Atlas Cloud

```go
m := ai.New("atlascloud",
    ai.WithAPIKey("your-key"),
    ai.WithModel("llama-3.3-70b"), // default
)
```

Default model: `llama-3.3-70b`
Default base URL: `https://api.atlascloud.ai`

Atlas Cloud is an enterprise AI infrastructure platform offering high-performance LLM APIs. It exposes an OpenAI-compatible chat completions endpoint with tool calling support.

## Auto-Detection

Use `AutoDetectProvider()` to detect the provider from a base URL:

```go
provider := ai.AutoDetectProvider("https://api.anthropic.com")
// Returns "anthropic"

m := ai.New(provider, ai.WithAPIKey("..."))
```

## Adding a New Provider

See the full **[AI Provider Integration Guide](../internal/website/docs/guides/ai-provider-guide.md)** for a step-by-step walkthrough, checklist, and design notes.

Quick summary:

1. Create `ai/yourprovider/yourprovider.go` implementing `ai.Model`.
2. Call `ai.Register("yourprovider", ...)` in `init()`.
3. Add tests in `ai/yourprovider/yourprovider_test.go`.
4. Users enable the provider with a blank import:

```go
import _ "go-micro.dev/v5/ai/yourprovider"
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
m := ai.New("openai", ai.WithAPIKey("..."))
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
