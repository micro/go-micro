---
layout: default
---

# CLI & Gateway Guide

The Go Micro CLI provides two gateway modes for accessing your microservices: development (`micro run`) and production (`micro server`). Both use the same underlying gateway architecture, ensuring consistent behavior across environments.

## Overview

```
                    ┌─────────────────────┐
                    │   HTTP Requests     │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Unified Gateway   │
                    │                     │
                    │  • Service Discovery│
                    │  • HTTP → RPC       │
                    │  • Web Dashboard    │
                    │  • Health Checks    │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Your Services     │
                    │  (via Registry)     │
                    └─────────────────────┘
```

## Quick Comparison

| Feature | `micro run` | `micro server` |
|---------|-------------|----------------|
| **Purpose** | Local development | Production API gateway |
| **Authentication** | Yes (default `admin`/`micro`) | Yes (default `admin`/`micro`) |
| **Process Management** | Yes (builds & runs services) | No (services run separately) |
| **Hot Reload** | Yes (watches file changes) | No |
| **Endpoint Scopes** | Yes (`/auth/scopes`) | Yes (`/auth/scopes`) |
| **Best For** | Coding, testing, iteration | Deployed environments |

## Development Mode: `micro run`

### Quick Start

```bash
# Create and run a service
micro new myservice
cd myservice
micro run
```

Open http://localhost:8080 - no login required!

### What You Get

- **Instant Gateway**: HTTP API at `/api/{service}/{method}`
- **Web Dashboard**: Browse and test services at `/`
- **Hot Reload**: Code changes trigger automatic rebuild
- **Authentication**: JWT auth with default credentials (`admin`/`micro`)
- **Scopes**: Endpoint access control via `/auth/scopes`

### Example Usage

```bash
# Start with hot reload
micro run

# Log in at http://localhost:8080 with admin/micro
# Or use a token for API calls:
curl -X POST http://localhost:8080/api/myservice/Handler.Call \
  -H "Authorization: Bearer <token>" \
  -d '{"name": "World"}'
```

### When to Use

- Writing new services
- Testing changes locally
- Debugging service interactions
- Testing auth and scopes before production

See [micro run guide](micro-run.md) for full details.

## Production Mode: `micro server`

### Quick Start

```bash
# Start your services separately (e.g., via systemd, docker)
./myservice &

# Start the gateway
micro server --address :8080
```

Open http://localhost:8080 and log in with `admin/micro`.

### What You Get

- **API Gateway**: Secure HTTP endpoint for all services
- **JWT Authentication**: Token-based access control
- **Web Dashboard**: Service management UI with login
- **User Management**: Create users and API tokens
- **Endpoint Scopes**: Fine-grained access control per endpoint
- **Production Ready**: Designed for deployed environments

### Authentication

All API calls require an `Authorization` header:

```bash
# Get a token (via web UI or login endpoint)
TOKEN="eyJhbGc..."

# Call a service with auth
curl -X POST http://localhost:8080/api/myservice/Handler.Call \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "World"}'
```

### Managing Users, Tokens & Scopes

1. **Log in**: Visit http://localhost:8080 → Enter `admin/micro`
2. **Create API Token**: Go to `/auth/tokens` → Generate token with scopes
3. **Set Endpoint Scopes**: Go to `/auth/scopes` → Restrict which endpoints require which scopes
4. **Use Token**: Copy and use in `Authorization: Bearer <token>` header

### When to Use

- Production deployments
- Staging environments
- Multi-team access (with auth)
- Public-facing APIs (with security)

## Gateway Features (Both Modes)

Both commands provide the same core gateway capabilities:

### 1. HTTP to RPC Translation

The gateway automatically converts HTTP requests to RPC calls:

```bash
POST /api/{service}/{method}
Content-Type: application/json

{"field": "value"}
```

Becomes an RPC call to:
- Service: `{service}`
- Method: `{method}`
- Payload: `{"field": "value"}`

### 2. Service Discovery

The gateway queries the registry (mdns, consul, etcd) to find services:

```bash
# List all services
curl http://localhost:8080/services

# Returns:
[
  {"name": "myservice", "endpoints": ["Handler.Call", "Handler.List"]},
  {"name": "users", "endpoints": ["Users.Create", "Users.Get"]}
]
```

Services register automatically when they start - no manual configuration needed!

### 3. Web Dashboard

Visit `/` in your browser to:

- Browse all registered services
- See available endpoints with request/response schemas
- Test endpoints with auto-generated forms
- View service health and status
- Read API documentation

### 4. Health Checks

```bash
# Aggregate health of all services
curl http://localhost:8080/health

# Kubernetes-style probes
curl http://localhost:8080/health/live   # Is gateway alive?
curl http://localhost:8080/health/ready  # Are services ready?
```

### 5. Dynamic Updates

The gateway automatically picks up:

- New services registering
- Services going offline
- Endpoint changes
- Version updates

No gateway restart needed!

### 6. Endpoint Scopes

Scopes provide fine-grained access control over which tokens can call which endpoints. Both `micro run` and `micro server` support scopes.

**Set up endpoint scopes:**

1. Visit `/auth/scopes` to see all discovered endpoints
2. Set required scopes for endpoints (e.g., `billing` on `payments.Payments.Charge`)
3. Use Bulk Set to apply scopes to all endpoints matching a pattern (e.g., `greeter.*`)

**Create scoped tokens:**

1. Visit `/auth/tokens` and create a token with matching scopes
2. A token with scope `billing` can call endpoints that require `billing`
3. A token with scope `*` bypasses all scope checks
4. Endpoints with no scopes set are open to any authenticated token

**Scopes are enforced on all call paths:**

- Direct API calls (`/api/{service}/{endpoint}`)
- MCP tool calls (`/api/mcp/call`)
- Agent playground tool invocations

The gateway uses `auth.Account` from the go-micro framework. The account's `Scopes` field carries the same `[]string` used by the framework's `wrapper/auth` package for service-level auth.

## Architecture Benefits

### Why Unified?

Previously, `micro run` and `micro server` had separate gateway implementations. This caused:

- ❌ Duplicated code (hard to maintain)
- ❌ Feature lag (improvements didn't benefit both)
- ❌ Inconsistent behavior between dev and prod

The unified gateway means:

- ✅ Single codebase for both commands
- ✅ Identical HTTP API in dev and production
- ✅ New features benefit both modes automatically
- ✅ Easier testing and maintenance

### What Changed for Users?

From a user perspective:

- `micro run` and `micro server` both have auth enabled
- Both use the same JWT authentication and scopes system
- API endpoints are unchanged
- Web UI is identical

The unification is internal - your code keeps working.

## Common Patterns

### Local Development → Production

```bash
# 1. Develop locally without auth
micro run
# Test: curl http://localhost:8080/api/...

# 2. Build for production
go build -o myservice

# 3. Deploy services
./myservice &  # or via systemd, docker, k8s

# 4. Start gateway with auth
micro server

# 5. Generate API token (via web UI)
# Use token in production API calls
```

### Multi-Service Development

```bash
# micro.mu
service api
    path ./api
    port 8081

service worker
    path ./worker
    port 8082
    depends api

service web
    path ./web
    port 8090
    depends api worker

# Start all with gateway
micro run
```

See [micro run guide](micro-run.md) for configuration details.

### API Gateway Deployment

Deploy `micro server` as your API gateway in front of all services:

```
                Internet
                    │
            ┌───────▼────────┐
            │  micro server  │  :8080 (public)
            │   + JWT Auth   │
            └───────┬────────┘
                    │
        ┌───────────┼───────────┐
        │           │           │
    ┌───▼───┐   ┌──▼───┐   ┌──▼────┐
    │ users │   │ posts│   │comments│
    │ :8081 │   │ :8082│   │ :8083  │
    └───────┘   └──────┘   └────────┘
    (internal)  (internal)  (internal)
```

Only `micro server` needs public access - services can be internal.

## Programmatic Usage

You can also use the gateway in your own Go code:

```go
package main

import (
    "context"
    "log"
    "go-micro.dev/v5/cmd/micro/server"
    "go-micro.dev/v5/store"
)

func main() {
    // Start gateway with custom options
    gw, err := server.StartGateway(server.GatewayOptions{
        Address:     ":9000",
        AuthEnabled: true,  // Enable authentication
        Store:       store.DefaultStore,
        Context:     context.Background(),
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Gateway running on %s", gw.Addr())

    // Block until context is cancelled
    gw.Wait()
}
```

This gives you full control over gateway configuration in custom deployments.

## Troubleshooting

### Gateway starts but no services show

**Problem**: http://localhost:8080 shows empty service list

**Solution**:
1. Check services are running: `ps aux | grep myservice`
2. Verify registry: services must register via mdns/consul/etcd
3. Check logs: `~/micro/logs/` for service startup errors

### API calls return 404

**Problem**: `curl http://localhost:8080/api/myservice/Handler.Call` returns 404

**Solution**:
1. Visit http://localhost:8080/services to see registered endpoints
2. Check exact endpoint name (case-sensitive): `Handler.Call` vs `handler.call`
3. Ensure service is registered: `micro services` or check web UI

### Authentication errors

**Problem**: API returns `401 Unauthorized`

**Solution**:
1. Generate token: Visit http://localhost:8080/auth/tokens
2. Use header: `Authorization: Bearer <token>`
3. Check token not expired (24h default)
4. Verify user not deleted (tokens revoked on user deletion)

### Scope errors

**Problem**: API returns `403 Forbidden` with `insufficient scopes`

**Solution**:
1. Check which scopes the endpoint requires: Visit `/auth/scopes`
2. Ensure your token has a matching scope (check at `/auth/tokens`)
3. Use a token with `*` scope for full access
4. Clear scopes from the endpoint if it should be unrestricted

### Port already in use

**Problem**: `micro run` or `micro server` won't start

**Solution**:
```bash
# Check what's using port 8080
lsof -i :8080

# Use different port
micro run --address :9000
micro server --address :9000
```

## Next Steps

- [Getting Started](../getting-started.md) - Build your first service
- [micro run Guide](micro-run.md) - Full development workflow
- [Deployment Guide](../deployment.md) - Deploy to production
- [Architecture](../architecture.md) - How it works internally

## Need Help?

- **Issues**: [github.com/micro/go-micro/issues](https://github.com/micro/go-micro/issues)
- **Discord**: [discord.gg/jwTYuUVAGh](https://discord.gg/jwTYuUVAGh)
- **Docs**: [go-micro.dev/docs](https://go-micro.dev/docs)
