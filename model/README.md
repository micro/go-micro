# Model Package

The `model` package provides an abstraction layer for AI model providers, allowing the micro server to interact with different LLM providers (Anthropic Claude, OpenAI GPT, etc.) through a common interface.

## Architecture

The package uses a provider factory pattern with automatic registration:

- **Model Interface**: Defines the contract for all model providers
- **Provider Registration**: Providers register themselves via `init()` functions
- **Auto-detection**: Automatically detects provider from base URL
- **Extensibility**: New providers can be added without modifying existing code

## Usage

### Basic Usage

```go
import (
    "go-micro.dev/v5/model"
    _ "go-micro.dev/v5/model/anthropic"
    _ "go-micro.dev/v5/model/openai"
)

// Create a model provider
provider, err := model.New("openai")
if err != nil {
    log.Fatal(err)
}

// Build a request
tools := []model.Tool{
    {
        Name:        "get_user",
        Description: "Get user information",
        Properties: map[string]any{
            "id": map[string]any{
                "type": "string",
                "description": "User ID",
            },
        },
    },
}

requestBody, err := provider.BuildRequest("Hello, world!", "You are a helpful assistant", tools, nil)
if err != nil {
    log.Fatal(err)
}

// Make API call and parse response
// ... HTTP call logic ...
response, err := provider.ParseResponse(responseBody)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Reply:", response.Reply)
fmt.Println("Tool calls:", len(response.ToolCalls))
```

### Auto-detection

```go
// Automatically detect provider from URL
provider := model.AutoDetectProvider("https://api.anthropic.com")
// Returns "anthropic"

provider = model.AutoDetectProvider("https://api.openai.com")
// Returns "openai"
```

## Supported Providers

### Anthropic Claude

- **Provider Name**: `anthropic`
- **Default Model**: `claude-sonnet-4-20250514`
- **Default Base URL**: `https://api.anthropic.com`
- **API Endpoint**: `/v1/messages`
- **Authentication**: `x-api-key` header

### OpenAI GPT

- **Provider Name**: `openai`
- **Default Model**: `gpt-4o`
- **Default Base URL**: `https://api.openai.com`
- **API Endpoint**: `/v1/chat/completions`
- **Authentication**: `Authorization: Bearer <token>` header

## Adding a New Provider

To add a new model provider:

1. Create a new directory under `model/` (e.g., `model/mymodel/`)
2. Implement the `model.Model` interface
3. Register your provider in an `init()` function:

```go
package mymodel

import "go-micro.dev/v5/model"

func init() {
    model.Register("mymodel", func(opts model.Options) model.Model {
        return NewProvider(opts)
    })
}

type Provider struct {
    options model.Options
}

func NewProvider(opts model.Options) *Provider {
    return &Provider{options: opts}
}

// Implement all Model interface methods...
func (p *Provider) Name() string { return "mymodel" }
func (p *Provider) DefaultModel() string { return "my-model-v1" }
// ... etc
```

4. Import your provider in the server:

```go
import _ "go-micro.dev/v5/model/mymodel"
```

## Interface Methods

### Core Methods

- `Name()` - Returns the provider name
- `DefaultModel()` - Returns the default model name
- `DefaultBaseURL()` - Returns the default API base URL

### Request Building

- `BuildRequest(prompt, systemPrompt, tools, messages)` - Builds initial request
- `BuildFollowUpRequest(prompt, systemPrompt, response, toolResults)` - Builds follow-up request with tool results

### Response Parsing

- `ParseResponse(body)` - Parses initial response
- `ParseFollowUpResponse(body)` - Parses follow-up response

### Configuration

- `SetAuthHeaders(headers, apiKey)` - Sets provider-specific auth headers
- `GetAPIEndpoint(baseURL)` - Returns full API endpoint URL

## Types

### Tool

Represents a tool/function that can be called by the model:

```go
type Tool struct {
    Name         string         // LLM-safe name
    OriginalName string         // Original name
    Description  string         // Tool description
    Properties   map[string]any // JSON schema for parameters
}
```

### Response

Parsed response from a model:

```go
type Response struct {
    Reply      string      // Text reply
    ToolCalls  []ToolCall  // Tool calls requested
    RawContent any         // Provider-specific raw content
}
```

### ToolCall

Request to call a tool:

```go
type ToolCall struct {
    ID    string         // Tool call ID
    Name  string         // Tool name
    Input map[string]any // Tool arguments
}
```

### ToolResult

Result of a tool execution:

```go
type ToolResult struct {
    ID      string // Tool call ID
    Content string // Result (JSON string)
}
```

## Testing

Run tests for all providers:

```bash
go test ./model/...
```

Run tests for a specific provider:

```bash
go test ./model/anthropic/
go test ./model/openai/
```
