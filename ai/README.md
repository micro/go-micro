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

## Auto-Detection

Use `AutoDetectProvider()` to detect the provider from a base URL:

```go
provider := ai.AutoDetectProvider("https://api.anthropic.com")
// Returns "anthropic"

m := ai.New(provider, ai.WithAPIKey("..."))
```

## Adding a New Provider

1. Create a new package under `ai/`:

```go
package myprovider

import "go-micro.dev/v5/ai"

func init() {
    ai.Register("myprovider", func(opts ...ai.Option) ai.Model {
        return NewProvider(opts...)
    })
}

type Provider struct {
    opts ai.Options
}

func NewProvider(opts ...ai.Option) *Provider {
    options := ai.NewOptions(opts...)
    // Set defaults
    if options.Model == "" {
        options.Model = "my-default-model"
    }
    if options.BaseURL == "" {
        options.BaseURL = "https://api.myprovider.com"
    }
    return &Provider{opts: options}
}

func (p *Provider) Init(opts ...ai.Option) error {
    for _, o := range opts {
        o(&p.opts)
    }
    return nil
}

func (p *Provider) Options() ai.Options {
    return p.opts
}

func (p *Provider) String() string {
    return "myprovider"
}

func (p *Provider) Generate(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (*ai.Response, error) {
    // Implement your provider logic
    // - Build API request
    // - Make HTTP call
    // - Parse response
    // - Handle tools if ToolHandler is set
    return &ai.Response{}, nil
}

func (p *Provider) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
    return nil, fmt.Errorf("streaming not implemented")
}
```

2. Import your provider:

```go
import _ "go-micro.dev/v5/ai/myprovider"
```

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
