# Model Context Protocol (MCP)

Go Micro provides built-in support for the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/), enabling AI agents like Claude to discover and interact with your microservices as tools.

## Overview

MCP gateway automatically exposes your microservices as AI-accessible tools through:
- **Automatic service discovery** via the registry
- **Dynamic tool generation** from service endpoints
- **Stdio transport** for local AI tools (Claude Code, etc.)
- **HTTP/SSE transport** for web-based agents
- **Automatic documentation extraction** from Go comments

## Quick Start

### 1. Add Documentation to Your Service

Simply write Go doc comments on your handler methods:

```go
package main

import (
    "context"
    "go-micro.dev/v5"
)

type GreeterService struct{}

// SayHello greets a person by name. Returns a friendly greeting message.
//
// @example {"name": "Alice"}
func (g *GreeterService) SayHello(ctx context.Context, req *HelloRequest, rsp *HelloResponse) error {
    rsp.Message = "Hello " + req.Name
    return nil
}

type HelloRequest struct {
    Name string `json:"name" description:"Person's name to greet"`
}

type HelloResponse struct {
    Message string `json:"message" description:"Greeting message"`
}

func main() {
    service := micro.NewService(
        micro.Name("greeter"),
    )

    service.Init()

    // Register handler - docs extracted automatically from comments!
    handler := service.Server().NewHandler(new(GreeterService))
    service.Server().Handle(handler)

    service.Run()
}
```

**That's it!** Documentation is automatically extracted from your Go comments.

### 2. Start the MCP Server

#### Option A: Stdio Transport (for Claude Code)

```bash
# Start your service
go run main.go

# In another terminal, start MCP server with stdio
micro mcp serve
```

Add to Claude Code config (\`~/.claude/claude_desktop_config.json\`):

```json
{
  "mcpServers": {
    "go-micro": {
      "command": "micro",
      "args": ["mcp", "serve"]
    }
  }
}
```

#### Option B: HTTP Transport (for web agents)

Start MCP gateway with HTTP/SSE:

```bash
micro mcp serve --address :3000
```

Access tools at \`http://localhost:3000/mcp/tools\`

### 3. Use Your Service with AI

Claude can now discover and call your service:

```
User: "Say hello to Bob using the greeter service"

Claude: [calls greeter.GreeterService.SayHello with {"name": "Bob"}]
       "Hello Bob"
```

## Features

### Automatic Documentation Extraction

Go Micro **automatically** extracts documentation from your handler method comments at registration time. No extra code needed!

For complete documentation details, see the [gateway/mcp package documentation](https://github.com/micro/go-micro/tree/master/gateway/mcp).

### Authentication & Scopes for MCP Tools

MCP tool calls go through the same authentication and scope enforcement as regular API calls. This means you can control which tokens (and therefore which users, services, or AI agents) can invoke which tools.

#### Restricting MCP Tool Access

1. **Set endpoint scopes** — Visit `/auth/scopes` and set required scopes on service endpoints. For example, set `internal` on `billing.Billing.Charge` to restrict it.

2. **Create scoped tokens** — Visit `/auth/tokens` and create tokens with specific scopes:
   - A token with scope `internal` can call endpoints requiring `internal`
   - A token with scope `*` has unrestricted access (admin)
   - A token with no matching scope gets `403 Forbidden`

3. **Use the token** — Pass it in the `Authorization` header for API/MCP calls:

```bash
# List available MCP tools (requires valid token)
curl http://localhost:8080/api/mcp/tools \
  -H "Authorization: Bearer <token>"

# Call a specific tool (scope-checked)
curl -X POST http://localhost:8080/api/mcp/call \
  -H "Authorization: Bearer <token>" \
  -d '{"tool":"greeter.GreeterService.SayHello","input":{"name":"World"}}'
```

#### Common MCP Token Patterns

| Use Case | Token Scopes | What It Can Do |
|----------|-------------|----------------|
| Internal tooling | `internal` | Call endpoints tagged with `internal` scope |
| Production AI agent | `greeter, users` | Only call greeter and user service endpoints |
| Admin / debugging | `*` | Full access to all tools |
| Read-only agent | `readonly` | Call endpoints tagged with `readonly` scope |

#### Agent Playground

The agent playground at `/agent` uses the logged-in user's session token. Scope checks apply based on the scopes of the user's account. The default `admin` user has `*` scope (full access).

### MCP Command Line

The \`micro mcp\` command provides tools for working with MCP:

```bash
# Start MCP server (stdio by default)
micro mcp serve

# Start with HTTP transport
micro mcp serve --address :3000

# List available tools
micro mcp list

# Test a specific tool
micro mcp test greeter.GreeterService.SayHello
```

### Transport Options

- **Stdio** - For local AI tools (Claude Code, recommended)
- **HTTP/SSE** - For web-based agents

See examples for complete usage.

## Examples

See \`examples/mcp/documented\` for a complete working example.

## Learn More

- [MCP Specification](https://modelcontextprotocol.io/)
- [Full Documentation Guide](https://github.com/micro/go-micro/blob/master/gateway/mcp/DOCUMENTATION.md)
- [Examples](https://github.com/micro/go-micro/tree/master/examples/mcp)

