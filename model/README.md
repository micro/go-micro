# Model Package

The `model` package provides a simple, high-level interface for AI model providers like Anthropic Claude and OpenAI GPT.

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
    "go-micro.dev/v5/model"
    _ "go-micro.dev/v5/model/anthropic"
    _ "go-micro.dev/v5/model/openai"
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
Default base URL: `https://api.openai.com`

## Auto-Detection

Use `AutoDetectProvider()` to detect the provider from a base URL:

```go
provider := model.AutoDetectProvider("https://api.anthropic.com")
// Returns "anthropic"

m := model.New(provider, model.WithAPIKey("..."))
```

## Adding a New Provider

1. Create a new package under `model/`:

```go
package myprovider

import "go-micro.dev/v5/model"

func init() {
    model.Register("myprovider", func(opts ...model.Option) model.Model {
        return NewProvider(opts...)
    })
}

type Provider struct {
    opts model.Options
}

func NewProvider(opts ...model.Option) *Provider {
    options := model.NewOptions(opts...)
    // Set defaults
    if options.Model == "" {
        options.Model = "my-default-model"
    }
    if options.BaseURL == "" {
        options.BaseURL = "https://api.myprovider.com"
    }
    return &Provider{opts: options}
}

func (p *Provider) Init(opts ...model.Option) error {
    for _, o := range opts {
        o(&p.opts)
    }
    return nil
}

func (p *Provider) Options() model.Options {
    return p.opts
}

func (p *Provider) String() string {
    return "myprovider"
}

func (p *Provider) Generate(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (*model.Response, error) {
    // Implement your provider logic
    // - Build API request
    // - Make HTTP call
    // - Parse response
    // - Handle tools if ToolHandler is set
    return &model.Response{}, nil
}

func (p *Provider) Stream(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (model.Stream, error) {
    return nil, fmt.Errorf("streaming not implemented")
}
```

2. Import your provider:

```go
import _ "go-micro.dev/v5/model/myprovider"
```

## Comparison with Other Packages

The model package follows the same patterns as other go-micro packages:

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

**Model:**
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
go test ./model/...
```

## Examples

See the [server implementation](../../cmd/micro/server/server.go) for a complete example of using the model package with tool execution.
