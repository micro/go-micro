---
layout: default
title: MCP Gateway
---

The MCP (Model Context Protocol) gateway automatically exposes your go-micro services as AI-accessible tools.

### Features

- **Automatic Service Discovery**: Queries the registry and exposes all service endpoints as MCP tools
- **Dynamic Updates**: Watches for service changes and updates tool list automatically
- **Multiple Transports**: Supports both stdio (for Claude Code) and HTTP/SSE (for web clients)
- **Zero Configuration**: Works out of the box with your existing services
- **Type-Safe**: Converts service schemas to JSON Schema for MCP

### Quick Start

#### Add to Existing Service

```go
package main

import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/gateway/mcp"
)

func main() {
    service := micro.NewService(micro.Name("myservice"))
    service.Init()

    // Add MCP gateway in 3 lines
    go mcp.Serve(mcp.Options{
        Registry: service.Options().Registry,
        Address:  ":3000",
    })

    service.Run()
}
```

#### Standalone Gateway

```go
package main

import (
    "go-micro.dev/v5/gateway/mcp"
    "go-micro.dev/v5/registry/mdns"
)

func main() {
    // Standalone MCP gateway
    // Discovers and exposes all services in registry
    mcp.ListenAndServe(":3000", mcp.Options{
        Registry: mdns.NewRegistry(),
    })
}
```

### Usage with Claude Code

Start your services with MCP gateway:

```bash
go run main.go
```

The MCP server will be available at `http://localhost:3000`. Claude Code or other MCP clients can connect and call your services as tools.

### API Endpoints

When using HTTP transport:

- `GET /mcp/tools` - List all available tools
- `POST /mcp/call` - Execute a tool (make RPC call)
- `GET /health` - Gateway health status

### Options

```go
type Options struct {
    // Registry for service discovery (required)
    Registry registry.Registry

    // Address for HTTP/SSE transport (e.g., ":3000")
    // Leave empty for stdio transport
    Address string

    // Client for RPC calls (optional, defaults to client.DefaultClient)
    Client client.Client

    // Context for cancellation (optional)
    Context context.Context

    // Logger for debug output (optional)
    Logger *log.Logger

    // AuthFunc validates requests (optional)
    AuthFunc func(r *http.Request) error
}
```

### With Authentication

```go
mcp.Serve(mcp.Options{
    Registry: registry.DefaultRegistry,
    Address:  ":3000",
    AuthFunc: func(r *http.Request) error {
        token := r.Header.Get("Authorization")
        if token != "Bearer secret" {
            return errors.New("unauthorized")
        }
        return nil
    },
})
```

### Docker Compose Example

```yaml
version: '3.8'

services:
  users:
    build: ./users
    environment:
      - MICRO_REGISTRY=mdns

  posts:
    build: ./posts
    environment:
      - MICRO_REGISTRY=mdns

  mcp-gateway:
    build: ./mcp-gateway
    ports:
      - "3000:3000"
    environment:
      - MICRO_REGISTRY=mdns
```

### How It Works

1. **Service Discovery**: Gateway queries your registry (mdns/consul/etcd)
2. **Tool Generation**: Each service endpoint becomes an MCP tool
3. **Schema Conversion**: Request/response types → JSON Schema
4. **RPC Translation**: MCP tool calls → go-micro RPC calls
5. **Dynamic Updates**: New services automatically appear as tools

### Tool Format

Services are exposed in this format:

```json
{
  "name": "users.Users.Create",
  "description": "Call Users.Create on users service",
  "inputSchema": {
    "type": "object",
    "properties": {
      "email": {"type": "string"},
      "name": {"type": "string"}
    }
  }
}
```

### Why MCP?

MCP makes your microservices **AI-native**:

- ✅ Claude can directly call your services
- ✅ No manual API wrappers needed
- ✅ No OpenAPI specs to maintain
- ✅ Services automatically become AI tools
- ✅ Perfect for AI assistants, debugging, operations

### Use Cases

- **AI Assistants**: Let Claude query your services
- **Debugging**: "Why is user 123's order failing?" → Claude investigates
- **Operations**: "Scale the worker service" → Claude calls admin APIs
- **Customer Support**: AI checks account status by calling your services

### License

Apache 2.0
