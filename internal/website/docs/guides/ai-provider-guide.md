# Adding an AI Provider to Go Micro

This guide walks you through implementing a new AI model provider for
go-micro's `ai` package. After following these steps your provider will
be available via `ai.New("yourprovider")` and automatically usable by the
MCP gateway, the agent playground, and any service that calls
`service.Model()`.

## Overview

The `ai` package uses the same plugin pattern as the rest of go-micro:
define an interface, register an implementation, and let users swap
providers with a single import. All providers live under `ai/<name>/`.

**Files you will create:**

```
ai/
└── yourprovider/
    ├── yourprovider.go       # Provider implementation
    └── yourprovider_test.go  # Unit tests
```


## Discover registered provider capabilities

Go Micro exposes the provider interfaces registered in the current build, so
runtime tooling and docs can report what is actually available after blank
imports are linked in:

```go
for _, row := range ai.CapabilityRows() {
    fmt.Printf("%s: chat=%t image=%t video=%t stream=%t tool_stream=%t\n", row.Provider, row.Model, row.Image, row.Video, row.Stream, row.ToolStream)
}
```

The built-in providers currently register these capability interfaces:

| Provider | Chat/text (`ai.Model`) | Image (`ai.ImageModel`) | Video (`ai.VideoModel`) | Streaming (`ai.Stream`) | Tool streaming |
| --- | --- | --- | --- | --- | --- |
| `anthropic` | Yes | No | No | Yes | Yes |
| `atlascloud` | Yes | Yes | Yes | Yes | No |
| `gemini` | Yes | No | No | Yes | No |
| `groq` | Yes | No | No | Yes | Yes |
| `minimax` | Yes | No | No | Yes | Yes |
| `mistral` | Yes | No | No | Yes | Yes |
| `ollama` | Yes | No | No | Yes | Yes |
| `openai` | Yes | Yes | No | Yes | Yes |
| `together` | Yes | No | No | Yes | Yes |

## Step 1: Implement the `ai.Model` Interface

Every provider must satisfy `ai.Model`:

```go
type Model interface {
    Init(...Option) error
    Options() Options
    Generate(ctx context.Context, req *Request, opts ...GenerateOption) (*Response, error)
    Stream(ctx context.Context, req *Request, opts ...GenerateOption) (Stream, error)
    String() string
}
```

### Skeleton

Create `ai/yourprovider/yourprovider.go`:

```go
package yourprovider

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"

    "go-micro.dev/v6/ai"
)

func init() {
    ai.Register("yourprovider", func(opts ...ai.Option) ai.Model {
        return NewProvider(opts...)
    })
}

type Provider struct {
    opts ai.Options
}

func NewProvider(opts ...ai.Option) *Provider {
    options := ai.NewOptions(opts...)
    if options.Model == "" {
        options.Model = "your-default-model"
    }
    if options.BaseURL == "" {
        options.BaseURL = "https://api.yourprovider.com"
    }
    return &Provider{opts: options}
}

func (p *Provider) Init(opts ...ai.Option) error {
    for _, o := range opts {
        o(&p.opts)
    }
    return nil
}

func (p *Provider) Options() ai.Options { return p.opts }
func (p *Provider) String() string      { return "yourprovider" }
```

### `Generate`

`Generate` is the core method. It must:

1. Convert `req.Tools` into the provider's native tool format.
2. Send the request to the provider API.
3. Parse the response into `ai.Response` (text in `Reply`, tool calls in
   `ToolCalls`).
4. If `p.opts.ToolHandler` is set **and** there are tool calls, execute
   each tool and make a follow-up API call to get the final answer in
   `Answer`.

```go
func (p *Provider) Generate(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (*ai.Response, error) {
    // 1. Build provider-specific tool definitions
    var tools []map[string]any
    for _, t := range req.Tools {
        tools = append(tools, map[string]any{
            // Map to your provider's schema
            "name":        t.Name,
            "description": t.Description,
            "parameters":  map[string]any{
                "type":       "object",
                "properties": t.Properties,
            },
        })
    }

    // 2. Build the API request body
    apiReq := map[string]any{
        "model":    p.opts.Model,
        "messages": []map[string]any{
            {"role": "system", "content": req.SystemPrompt},
            {"role": "user", "content": req.Prompt},
        },
    }
    if len(tools) > 0 {
        apiReq["tools"] = tools
    }

    // 3. Call the API
    resp, rawMsg, err := p.callAPI(ctx, apiReq)
    if err != nil {
        return nil, err
    }

    // 4. No tool calls → return immediately
    if len(resp.ToolCalls) == 0 {
        return resp, nil
    }

    // 5. Execute tools and follow up
    if p.opts.ToolHandler != nil {
        // ... build follow-up messages with tool results ...
        followUpResp, _, err := p.callAPI(ctx, followUpReq)
        if err == nil && followUpResp.Reply != "" {
            resp.Answer = followUpResp.Reply
        }
    }

    return resp, nil
}
```

### `Stream`

If streaming is not supported yet, return a clear error:

```go
func (p *Provider) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
    return nil, fmt.Errorf("streaming not yet implemented for yourprovider")
}
```

### API Helper

Use `net/http` directly — no external SDK needed:

```go
func (p *Provider) callAPI(ctx context.Context, req map[string]any) (*ai.Response, map[string]any, error) {
    reqBody, err := json.Marshal(req)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    apiURL := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/chat/completions"
    httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(reqBody))
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create request: %w", err)
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+p.opts.APIKey)

    httpResp, err := http.DefaultClient.Do(httpReq)
    if err != nil {
        return nil, nil, fmt.Errorf("API request failed: %w", err)
    }
    defer httpResp.Body.Close()

    respBody, _ := io.ReadAll(httpResp.Body)
    if httpResp.StatusCode != 200 {
        return nil, nil, fmt.Errorf("API error (%s): %s", httpResp.Status, string(respBody))
    }

    // Parse your provider's response format into ai.Response
    // ...
}
```

## Step 2: Write Tests

Create `ai/yourprovider/yourprovider_test.go`. At minimum test:

- **`String()`** returns the correct name.
- **`Init()`** applies options.
- **Default values** are set when no options are provided.
- **`Generate()` without API key** returns an error.
- **`Stream()` not implemented** returns an error.

```go
package yourprovider

import (
    "context"
    "testing"

    "go-micro.dev/v6/ai"
)

func TestProvider_String(t *testing.T) {
    p := NewProvider()
    if p.String() != "yourprovider" {
        t.Errorf("got %q, want %q", p.String(), "yourprovider")
    }
}

func TestProvider_Defaults(t *testing.T) {
    p := NewProvider()
    opts := p.Options()
    if opts.Model != "your-default-model" {
        t.Errorf("default model = %q, want %q", opts.Model, "your-default-model")
    }
    if opts.BaseURL != "https://api.yourprovider.com" {
        t.Errorf("default base URL = %q", opts.BaseURL)
    }
}

func TestProvider_Init(t *testing.T) {
    p := NewProvider()
    if err := p.Init(ai.WithModel("custom"), ai.WithAPIKey("key")); err != nil {
        t.Fatalf("Init: %v", err)
    }
    if p.Options().Model != "custom" {
        t.Errorf("model not updated")
    }
}

func TestProvider_Generate_NoAPIKey(t *testing.T) {
    p := NewProvider()
    _, err := p.Generate(context.Background(), &ai.Request{Prompt: "hi"})
    if err == nil {
        t.Error("expected error without API key")
    }
}

func TestProvider_Stream_NotImplemented(t *testing.T) {
    p := NewProvider()
    _, err := p.Stream(context.Background(), &ai.Request{Prompt: "hi"})
    if err == nil {
        t.Error("expected error for unimplemented streaming")
    }
}
```

Run:

```bash
go test ./ai/yourprovider/...
```

## Step 3: Register the Provider

The `init()` function in your package calls `ai.Register`. Users enable
your provider with a blank import:

```go
import _ "go-micro.dev/v6/ai/yourprovider"
```

Then use it:

```go
m := ai.New("yourprovider",
    ai.WithAPIKey("your-api-key"),
    ai.WithModel("your-model-name"),
)

resp, err := m.Generate(ctx, &ai.Request{
    Prompt:       "Hello!",
    SystemPrompt: "You are a helpful assistant",
})
```

## Step 4: Update the README

Add your provider to the **Supported AI Providers** section in the
project README.md. Follow the existing format:

```markdown
### YourProvider

```go
m := ai.New("yourprovider",
    ai.WithAPIKey("your-key"),
    ai.WithModel("your-default-model"),
)
```

Default model: `your-default-model`
Default base URL: `https://api.yourprovider.com`
```

Also add an entry in `ai/README.md` under "Supported Providers".

## Checklist

Before submitting your PR:

- [ ] `ai/yourprovider/yourprovider.go` implements `ai.Model`
- [ ] `init()` calls `ai.Register("yourprovider", ...)`
- [ ] `Generate()` handles tool calls via `ToolHandler` when set
- [ ] `ai/yourprovider/yourprovider_test.go` covers basics
- [ ] `go test ./ai/yourprovider/...` passes
- [ ] `go vet ./ai/yourprovider/...` is clean
- [ ] Provider added to `ai/README.md` under "Supported Providers"
- [ ] Provider added to project README.md under "Supported AI Providers"
- [ ] No new dependencies beyond `go-micro.dev/v6/ai` and stdlib (use
      `net/http` directly rather than an SDK)

## Design Notes

**Why `net/http` instead of an SDK?** Keeping providers dependency-free
means `go get go-micro.dev/v6` never pulls in heavy SDK trees. All
existing providers (Anthropic, OpenAI) use raw HTTP for the same reason.

**OpenAI-compatible APIs.** Many providers (Together, Groq, Fireworks,
Atlas Cloud, etc.) expose an OpenAI-compatible `/v1/chat/completions`
endpoint. In that case, users can often just use the `openai` provider
with `ai.WithBaseURL("https://api.yourprovider.com")`. A dedicated
provider package is only needed when the API differs or you want to set
provider-specific defaults.

**Tool call loop.** The current contract is one round of tool execution:
`Generate` calls tools via `ToolHandler`, feeds results back, and
returns the final answer. Multi-turn agentic loops are handled at a
higher level (e.g. the MCP gateway).

## Sponsorship

If you are an AI infrastructure company interested in becoming a
supported provider, we welcome both code contributions and sponsorships.
See the Supported AI Providers section in the project README for
current partners, and reach out via a GitHub issue or the Discord
community to discuss integration.
