---
title: "Model Context Protocol (MCP)"
---

Go Micro provides built-in support for the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/), enabling AI agents like Claude to discover and interact with your microservices as tools.

<img src="/images/generated/mcp-agent.jpg" alt="AI agent calling microservices via MCP" style="width: 100%; border-radius: 8px; margin: 1rem 0 1.5rem;" />

## Overview

MCP gateway automatically exposes your microservices as AI-accessible tools through:
- **Automatic service discovery** via the registry
- **Dynamic tool generation** from service endpoints
- **Stdio transport** for local AI tools (Claude Code, etc.)
- **Streamable-HTTP transport** (spec-compliant JSON-RPC 2.0 at `/mcp`) for browser MCP clients
- **WebSocket transport** at `/mcp/ws` for streaming agent frameworks
- **Legacy REST endpoints** at `/mcp/tools` and `/mcp/call` for simple tool access
- **Automatic documentation extraction** from Go comments

## Quick Start

### 1. Add Documentation to Your Service

Simply write Go doc comments on your handler methods:

```go
package main

import (
    "context"
    "go-micro.dev/v6"
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
    service := micro.NewService("greeter")
    service.Init()

    // Register handler - docs extracted automatically from comments!
    service.Handle(new(GreeterService))

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

For spec-compliant MCP clients (browser and desktop agents), use the streamable-HTTP endpoint at `http://localhost:3000/mcp` — see [Streamable-HTTP Transport](#streamable-http-transport-browser-mcp-clients) below.

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
curl http://localhost:8080/mcp/tools \
  -H "Authorization: Bearer <token>"

# Call a specific tool (scope-checked)
curl -X POST http://localhost:8080/mcp/call \
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

The MCP gateway serves four transports from one HTTP server (or stdio when no address is set):

- **Stdio** - For local AI tools (Claude Code, recommended). Start with `micro mcp serve`.
- **Streamable-HTTP** - Spec-compliant JSON-RPC 2.0 at `/mcp`. The endpoint for browser MCP clients; CORS is enabled so it works without a proxy.
- **WebSocket** - Bidirectional streaming at `/mcp/ws` for streaming agent frameworks.
- **Legacy REST** - Tool listing and calls at `/mcp/tools` and `/mcp/call`.

See examples for complete usage.

### Streamable-HTTP Transport (Browser MCP Clients)

Point any spec-compliant MCP client (Claude desktop, browser agents) at `http://localhost:3000/mcp`. The client SDK handles the handshake; the endpoint behaves like this:

1. **`POST /mcp`** with `initialize` mints a session, returned in the `Mcp-Session-Id` response header:

```bash
curl -i -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"my-agent","version":"1.0"}}}'

# Mcp-Session-Id: <session-id>   <- echo this header on later requests
```

2. **`POST /mcp`** with `tools/list` and `tools/call`, echoing the session id:

```bash
curl -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: <session-id>" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'

curl -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: <session-id>" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"greeter.GreeterService.SayHello","arguments":{"name":"World"}}}'
```

3. **`GET /mcp`** opens a `text/event-stream` for server→client messages; **`DELETE /mcp`** terminates the session. JSON-RPC notifications (no `id`) return `202 Accepted` with no body.

Supported methods: `initialize`, `ping`, `tools/list`, `tools/call`. Tool calls share the same auth, scope, rate-limit, circuit-breaker, and x402 payment pipeline as the REST endpoints, so scopes set in `/auth/scopes` are enforced identically.

### Gateway Independence

The MCP gateway is its own server, independent of the HTTP API gateway. The CLI (`micro gateway --mcp-address :3000`, or `micro run --mcp-address :3000`) starts both servers and shuts them down together; library users start each explicitly with `mcp.NewServer`.

## Examples

See \`examples/mcp/documented\` for a complete working example.

## Learn More

- [MCP Specification](https://modelcontextprotocol.io/)
- [Full Documentation Guide](https://github.com/micro/go-micro/blob/master/gateway/mcp/DOCUMENTATION.md)
- [Examples](https://github.com/micro/go-micro/tree/master/examples/mcp)
