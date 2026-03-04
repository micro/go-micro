---
layout: default
title: Add MCP to Existing Services
---

# Add MCP to Existing Services

You have a working go-micro service and want to make it accessible to AI agents via MCP. This guide covers the three approaches, from simplest to most flexible.

## Option 1: One-Line Setup (Recommended)

Add a single option to your service constructor:

```go
import "go-micro.dev/v5/gateway/mcp"

func main() {
    service := micro.New("myservice",
        mcp.WithMCP(":3001"),  // Add this line
    )
    service.Init()
    // ... register handlers as before
    service.Run()
}
```

That's it. Your service now exposes all registered handlers as MCP tools at `http://localhost:3001/mcp/tools`.

## Option 2: Standalone MCP Gateway

If you want the MCP gateway to run separately from your services (e.g., in production with multiple services):

```go
import "go-micro.dev/v5/gateway/mcp"

// Start MCP gateway alongside your service
go mcp.ListenAndServe(":3001", mcp.Options{
    Registry: service.Options().Registry,
})
```

This discovers all services in the registry and exposes them as tools.

## Option 3: CLI (No Code Changes)

If you don't want to modify your service code at all:

```bash
# Start your service normally
go run .

# In another terminal, start the MCP gateway
micro mcp serve --address :3001
```

The CLI approach uses the same registry to discover running services.

## Improving Agent Experience

Once MCP is enabled, improve how agents interact with your service by adding documentation.

### Step 1: Add Doc Comments

Before:
```go
func (s *Users) Get(ctx context.Context, req *GetRequest, rsp *GetResponse) error {
```

After:
```go
// Get retrieves a user by their unique ID. Returns the full user profile
// including email, display name, and account status.
//
// @example {"id": "user-123"}
func (s *Users) Get(ctx context.Context, req *GetRequest, rsp *GetResponse) error {
```

The MCP gateway automatically extracts these comments and presents them to agents as tool descriptions.

### Step 2: Add Struct Tag Descriptions

```go
type GetRequest struct {
    ID string `json:"id" description:"User ID in UUID format"`
}

type GetResponse struct {
    Name   string `json:"name" description:"Display name"`
    Email  string `json:"email" description:"Primary email address"`
    Active bool   `json:"active" description:"Whether the account is active"`
}
```

### Step 3: Add Auth Scopes (Optional)

Restrict which agents can call which endpoints:

```go
handler := service.Server().NewHandler(
    new(Users),
    server.WithEndpointScopes("Users.Delete", "users:admin"),
    server.WithEndpointScopes("Users.Get", "users:read"),
)
```

Then configure the MCP gateway with auth:

```go
mcp.ListenAndServe(":3001", mcp.Options{
    Registry: service.Options().Registry,
    Auth:     authProvider,
    Scopes: map[string][]string{
        "myservice.Users.Delete": {"users:admin"},
        "myservice.Users.Get":    {"users:read"},
    },
})
```

## Using with Claude Code

Once your service is running with MCP, connect it to Claude Code:

```bash
# Option A: stdio transport (recommended for local dev)
micro mcp serve

# Option B: Add to Claude Code settings
```

```json
{
  "mcpServers": {
    "my-services": {
      "command": "micro",
      "args": ["mcp", "serve"]
    }
  }
}
```

## Verify It Works

```bash
# List all tools the MCP gateway exposes
curl http://localhost:3001/mcp/tools | jq

# Test a specific tool
curl -X POST http://localhost:3001/mcp/call \
  -H 'Content-Type: application/json' \
  -d '{"tool": "myservice.Users.Get", "arguments": {"id": "user-123"}}'
```

## What Doesn't Need to Change

- **Handler signatures** - No changes needed to your RPC handlers
- **Proto definitions** - Existing protos work as-is
- **Client code** - Services calling each other still use the normal RPC client
- **Tests** - Existing tests continue to work
- **Deployment** - Add a port for MCP, everything else stays the same

## Next Steps

- [Tool Descriptions Guide](../tool-descriptions.md) - Write better descriptions for agents
- [MCP Security Guide](../mcp-security.md) - Auth, scopes, and audit logging
- [Agent Patterns](../agent-patterns.md) - Architecture patterns for agent integration
