---
layout: default
---

# MCP Security Guide

This guide covers how to secure your MCP gateway for production use, including authentication, per-tool scopes, rate limiting, and audit logging.

## Overview

The MCP gateway provides four layers of security:

1. **Authentication** - Verify the caller's identity via bearer tokens
2. **Scopes** - Control which tools each token can access
3. **Rate Limiting** - Prevent abuse with per-tool rate limits
4. **Audit Logging** - Record every tool call for compliance and debugging

## Authentication

### Bearer Token Auth

The MCP gateway uses bearer token authentication. Tokens are validated by the configured `auth.Auth` provider.

```go
import (
    "go-micro.dev/v5/gateway/mcp"
    "go-micro.dev/v5/auth"
)

gateway := mcp.ListenAndServe(":3000", mcp.Options{
    Registry: service.Options().Registry,
    Auth:     authProvider, // auth.Auth implementation
})
```

Agents pass tokens in the `Authorization` header:

```bash
curl -X POST http://localhost:3000/mcp/call \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"tool": "tasks.TaskService.Create", "input": {"title": "New task"}}'
```

### Using micro run / micro server

When using `micro run` or `micro server`, authentication is handled automatically:

- **Development mode (`micro run`):** Auth is disabled by default for easy development
- **Production mode (`micro server`):** JWT auth is enabled with user management at `/auth`

Create tokens with specific scopes via the dashboard at `/auth/tokens`.

## Per-Tool Scopes

Scopes control which tools a token can access. There are two ways to set scopes.

### Service-Level Scopes

Set scopes when registering your handler. These travel with the service through the registry:

```go
handler := service.Server().NewHandler(
    new(TaskService),
    server.WithEndpointScopes("TaskService.Get", "tasks:read"),
    server.WithEndpointScopes("TaskService.List", "tasks:read"),
    server.WithEndpointScopes("TaskService.Create", "tasks:write"),
    server.WithEndpointScopes("TaskService.Update", "tasks:write"),
    server.WithEndpointScopes("TaskService.Delete", "tasks:admin"),
)
```

### Gateway-Level Scopes

Override or add scopes at the gateway without modifying services. Gateway scopes take precedence:

```go
mcp.ListenAndServe(":3000", mcp.Options{
    Registry: reg,
    Auth:     authProvider,
    Scopes: map[string][]string{
        "tasks.TaskService.Create": {"tasks:write"},
        "tasks.TaskService.Delete": {"tasks:admin"},
        "billing.Billing.Charge":   {"billing:admin"},
    },
})
```

### Scope Enforcement

When a tool is called:

1. Gateway checks if the tool has required scopes
2. If scopes are defined, the caller's token must include at least one matching scope
3. A token with scope `*` has unrestricted access (admin)
4. If no scopes are defined for a tool, any authenticated token can call it
5. Denied calls return `403 Forbidden`

### Common Scope Patterns

| Pattern | Use Case |
|---------|----------|
| `service:read` | Read-only access to a service |
| `service:write` | Create and update operations |
| `service:admin` | Delete and destructive operations |
| `*` | Full admin access (use sparingly) |
| `internal` | Internal-only tools not exposed to external agents |

### Token Examples

```
Token A: scopes=["tasks:read"]
  ✅ Can call TaskService.Get, TaskService.List
  ❌ Cannot call TaskService.Create, TaskService.Delete

Token B: scopes=["tasks:read", "tasks:write"]
  ✅ Can call Get, List, Create, Update
  ❌ Cannot call TaskService.Delete (needs tasks:admin)

Token C: scopes=["*"]
  ✅ Can call everything (admin)
```

## Rate Limiting

Prevent abuse with per-tool rate limiting using a token bucket algorithm:

```go
mcp.ListenAndServe(":3000", mcp.Options{
    Registry: reg,
    RateLimit: &mcp.RateLimitConfig{
        RequestsPerSecond: 10,  // Sustained rate
        Burst:             20,  // Allow bursts up to 20
    },
})
```

When the rate limit is exceeded, calls return `429 Too Many Requests`.

### Choosing Rate Limits

| Service Type | Requests/sec | Burst | Rationale |
|-------------|-------------|-------|-----------|
| Read-heavy API | 50 | 100 | High throughput, low cost |
| Write API | 10 | 20 | Moderate, prevents spam |
| Expensive operation | 2 | 5 | Protect downstream resources |
| Internal tool | 100 | 200 | Trusted callers, higher limits |

## Audit Logging

Record every tool call for compliance, debugging, and analytics:

```go
mcp.ListenAndServe(":3000", mcp.Options{
    Registry: reg,
    Auth:     authProvider,
    AuditFunc: func(record mcp.AuditRecord) {
        log.Printf("[AUDIT] tool=%s account=%s allowed=%v duration=%v err=%v",
            record.Tool,
            record.AccountID,
            record.Allowed,
            record.Duration,
            record.Error,
        )
    },
})
```

### AuditRecord Fields

| Field | Type | Description |
|-------|------|-------------|
| `Tool` | `string` | Full tool name (e.g., `tasks.TaskService.Create`) |
| `AccountID` | `string` | Caller's account ID from the auth token |
| `Scopes` | `[]string` | Scopes on the caller's token |
| `Allowed` | `bool` | Whether the call was permitted |
| `Duration` | `time.Duration` | How long the call took |
| `Error` | `error` | Error if the call failed |
| `TraceID` | `string` | UUID trace ID for correlation |

### Production Audit Logging

For production, send audit records to a structured logging system:

```go
AuditFunc: func(r mcp.AuditRecord) {
    // Structured JSON logging
    logger.Info("mcp_tool_call",
        "tool", r.Tool,
        "account", r.AccountID,
        "allowed", r.Allowed,
        "duration_ms", r.Duration.Milliseconds(),
        "trace_id", r.TraceID,
    )

    // Alert on denied calls
    if !r.Allowed {
        alerting.Notify("MCP access denied",
            "tool", r.Tool,
            "account", r.AccountID,
        )
    }
},
```

## Tracing

Every MCP tool call gets a UUID trace ID, propagated via metadata headers:

| Header | Description |
|--------|-------------|
| `Mcp-Trace-Id` | UUID for the tool call |
| `Mcp-Tool-Name` | Name of the tool called |
| `Mcp-Account-Id` | Caller's account ID |

These are available in your handler via context metadata:

```go
func (t *TaskService) Create(ctx context.Context, req *CreateRequest, rsp *CreateResponse) error {
    md, _ := metadata.FromContext(ctx)
    traceID := md["Mcp-Trace-Id"]
    log.Printf("Creating task, trace: %s", traceID)
    // ...
}
```

## Production Checklist

Before deploying MCP to production:

- [ ] **Auth enabled** - Configure an `auth.Auth` provider
- [ ] **Scopes defined** - Every write/delete endpoint has required scopes
- [ ] **Rate limits set** - Appropriate limits for each service type
- [ ] **Audit logging active** - All calls logged to a persistent store
- [ ] **HTTPS/TLS** - MCP gateway behind TLS termination
- [ ] **Token rotation** - Process for rotating compromised tokens
- [ ] **Monitoring** - Alerts on high error rates or denied calls
- [ ] **Testing** - Verified scope enforcement with `micro mcp test`

## Full Example

```go
package main

import (
    "log"

    "go-micro.dev/v5"
    "go-micro.dev/v5/auth"
    "go-micro.dev/v5/gateway/mcp"
    "go-micro.dev/v5/server"
)

func main() {
    service := micro.NewService(
        micro.Name("tasks"),
        micro.Address(":8081"),
    )
    service.Init()

    // Register handler with scopes
    handler := service.Server().NewHandler(
        &TaskService{tasks: make(map[string]*Task)},
        server.WithEndpointScopes("TaskService.Get", "tasks:read"),
        server.WithEndpointScopes("TaskService.Create", "tasks:write"),
        server.WithEndpointScopes("TaskService.Delete", "tasks:admin"),
    )
    service.Server().Handle(handler)

    // Start MCP gateway with full security
    go mcp.ListenAndServe(":3000", mcp.Options{
        Registry: service.Options().Registry,
        Auth:     service.Options().Auth,
        Scopes: map[string][]string{
            // Gateway-level overrides
            "billing.Billing.Charge": {"billing:admin"},
        },
        RateLimit: &mcp.RateLimitConfig{
            RequestsPerSecond: 10,
            Burst:             20,
        },
        AuditFunc: func(r mcp.AuditRecord) {
            log.Printf("[AUDIT] tool=%s account=%s allowed=%v duration=%v",
                r.Tool, r.AccountID, r.Allowed, r.Duration)
        },
    })

    service.Run()
}
```

## Next Steps

- [Building AI-Native Services](ai-native-services.md) - End-to-end tutorial
- [Tool Description Best Practices](tool-descriptions.md) - Write effective documentation
- [Agent Integration Patterns](agent-patterns.md) - Multi-agent architectures
