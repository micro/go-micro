# ADR-010: Unified Gateway Architecture

**Status:** Accepted
**Date:** 2026-02-11
**Authors:** Go Micro Team

## Context

Previously, the go-micro CLI had two separate gateway implementations:

1. **`micro run`** gateway (`cmd/micro/run/gateway/`) - Simple HTTP-to-RPC proxy for development
2. **`micro server`** gateway (`cmd/micro/server/`) - Production gateway with authentication, web UI, and API documentation

This duplication created several problems:

- **Code maintenance**: Gateway logic (HTTP-to-RPC translation, service discovery, health checks) was implemented twice
- **Feature parity**: Improvements to one gateway didn't automatically benefit the other
- **Complexity**: New features (like MCP integration) would need to be implemented twice
- **Testing burden**: Each gateway required separate testing

## Decision

We unified the gateway implementation by:

1. **Extracting reusable gateway module** (`cmd/micro/server/gateway.go`):
   - `GatewayOptions` struct for configuration
   - `StartGateway()` function that returns a `*Gateway` immediately
   - `RunGateway()` function that blocks until shutdown
   - Configurable authentication (enabled/disabled)

2. **Refactoring `micro server`**:
   - Gateway logic remains in `cmd/micro/server/`
   - `registerHandlers()` now uses instance-specific `*http.ServeMux` instead of global mux
   - Authentication middleware is conditional based on `GatewayOptions.AuthEnabled`
   - Auth routes only register when authentication is enabled

3. **Updating `micro run`**:
   - Removed duplicate gateway implementation (`cmd/micro/run/gateway/`)
   - Now calls `server.StartGateway()` with `AuthEnabled: false`
   - Retains process management and hot reload functionality

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Unified Gateway                         │
│              (cmd/micro/server/gateway.go)                  │
│                                                             │
│  • HTTP → RPC translation                                  │
│  • Service discovery via registry                          │
│  • Web UI (dashboard, logs, API docs)                      │
│  • Health checks                                           │
│  • Configurable authentication                             │
└─────────────────────────────────────────────────────────────┘
           ▲                              ▲
           │                              │
    ┌──────┴──────┐              ┌────────┴────────┐
    │  micro run  │              │  micro server   │
    │             │              │                 │
    │  + Process  │              │  + Auth enabled │
    │    mgmt     │              │  + JWT tokens   │
    │  + Hot      │              │  + Production   │
    │    reload   │              │                 │
    │  - No auth  │              │                 │
    └─────────────┘              └─────────────────┘
```

## Usage

### Development Mode (`micro run`)

```bash
# Start services with gateway (no auth)
micro run

# Gateway provides:
# - HTTP API at /api/{service}/{endpoint}
# - Web dashboard at /
# - No authentication required
```

### Production Mode (`micro server`)

```bash
# Start gateway with authentication
micro server --address :8080

# Gateway provides:
# - HTTP API at /api/{service}/{endpoint} (auth required)
# - Web dashboard with login
# - JWT-based authentication
# - User/token management UI
```

## Benefits

1. **Single Source of Truth**: Gateway logic lives in one place
2. **Automatic Feature Propagation**: New features (like MCP) added to the unified gateway benefit both commands
3. **Simplified Testing**: Test gateway once, works everywhere
4. **Reduced Code Size**: Eliminated ~300 lines of duplicate code
5. **Clear Separation**:
   - `micro server` = API gateway (HTTP + future MCP)
   - `micro run` = Development tool (gateway + process management + hot reload)

## Implementation Details

### GatewayOptions

```go
type GatewayOptions struct {
    Address     string        // Listen address (e.g., ":8080")
    AuthEnabled bool          // Enable JWT authentication
    Store       store.Store   // Storage for auth data
    Context     context.Context // Cancellation context
}
```

### Starting the Gateway

```go
// Non-blocking start
gw, err := server.StartGateway(server.GatewayOptions{
    Address:     ":8080",
    AuthEnabled: false,
})

// Blocking start
err := server.RunGateway(server.GatewayOptions{
    Address:     ":8080",
    AuthEnabled: true,
})
```

### Authentication

When `AuthEnabled: true`:
- Auth middleware checks JWT tokens on all requests
- Auth routes are registered: `/auth/login`, `/auth/logout`, `/auth/tokens`, `/auth/users`
- Web UI requires login
- API endpoints require `Authorization: Bearer <token>` header

When `AuthEnabled: false` (dev mode):
- No authentication middleware
- Auth routes are not registered
- All endpoints are publicly accessible

## Consequences

### Positive

- Easier to add new features (only implement once)
- Better code maintainability
- Consistent behavior between development and production
- Foundation for MCP integration

### Negative

- `cmd/micro/run` now depends on `cmd/micro/server` (acceptable for CLI tools)
- Slightly more complex initialization in `micro run` (but cleaner overall)

## Future Work

With unified gateway architecture, we can now add:

1. **MCP Integration**: Add `mcp.go` to server package, both commands get MCP support
2. **GraphQL API**: Single implementation serves both dev and prod
3. **gRPC Gateway**: Expose services via gRPC alongside HTTP
4. **API Versioning**: Consistent versioning strategy across all deployments

## References

- Original issue: Gateway duplication between `micro run` and `micro server`
- Implementation: PR #XXX (gateway unification)
- Related: ADR-001 (Plugin Architecture), ADR-009 (Progressive Configuration)
