# API Gateway

The `gateway/api` package provides HTTP API gateway functionality for go-micro services. It translates HTTP requests into RPC calls and serves a web dashboard for browsing and calling services.

## Features

- **HTTP to RPC translation** - Call microservices via HTTP
- **Web dashboard** - Browse and test services in the browser
- **Authentication** - Optional JWT-based auth
- **MCP integration** - Expose services to AI agents
- **Flexible configuration** - Use in dev or production
- **Service discovery** - Auto-detect services from registry

## Usage

### Basic Gateway

```go
package main

import (
    "context"
    "net/http"

    "go-micro.dev/v5/gateway/api"
)

func main() {
    // Create gateway with custom handler
    gw, err := api.New(api.Options{
        Address: ":8080",
        Context: context.Background(),
        HandlerRegistrar: func(mux *http.ServeMux) error {
            // Register your HTTP handlers
            mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
                w.Write([]byte("Hello from gateway"))
            })
            return nil
        },
    })
    if err != nil {
        panic(err)
    }

    // Block until shutdown
    gw.Wait()
}
```

### Gateway with MCP

```go
gw, err := api.New(api.Options{
    Address:    ":8080",
    MCPEnabled: true,
    MCPAddress: ":3000", // MCP on separate port
    HandlerRegistrar: registerHandlers,
})
```

### Gateway with Authentication

```go
gw, err := api.New(api.Options{
    Address:     ":8080",
    AuthEnabled: true, // Handler registrar should add auth middleware
    HandlerRegistrar: func(mux *http.ServeMux) error {
        // Register handlers with auth middleware
        return registerAuthenticatedHandlers(mux)
    },
})
```

### Blocking Mode

```go
// Run blocks until shutdown
err := api.Run(api.Options{
    Address: ":8080",
    HandlerRegistrar: registerHandlers,
})
```

## Options

```go
type Options struct {
    // Address to listen on (default: ":8080")
    Address string

    // AuthEnabled signals that authentication is required
    // The HandlerRegistrar should implement auth checks
    AuthEnabled bool

    // Context for cancellation (default: context.Background())
    Context context.Context

    // Logger for gateway messages (default: log.Default())
    Logger *log.Logger

    // HandlerRegistrar registers HTTP handlers on the mux
    HandlerRegistrar func(mux *http.ServeMux) error

    // MCPEnabled enables the MCP gateway
    MCPEnabled bool

    // MCPAddress is the address for MCP gateway (e.g., ":3000")
    MCPAddress string

    // Registry for service discovery (default: registry.DefaultRegistry)
    Registry registry.Registry
}
```

## Architecture

```
┌─────────────────────────────────────────┐
│         gateway/api Package              │
│  ┌────────────────────────────────────┐ │
│  │  Gateway                           │ │
│  │  - Manages HTTP server             │ │
│  │  - Calls HandlerRegistrar          │ │
│  │  - Starts MCP if enabled           │ │
│  └────────────────────────────────────┘ │
└─────────────────────────────────────────┘
               ↓ delegates to
┌─────────────────────────────────────────┐
│     HandlerRegistrar (user-provided)     │
│  ┌────────────────────────────────────┐ │
│  │  func(mux *http.ServeMux) error    │ │
│  │  - Registers routes                │ │
│  │  - Adds middleware (auth, etc.)    │ │
│  │  - Sets up templates               │ │
│  └────────────────────────────────────┘ │
└─────────────────────────────────────────┘
               ↓ uses
┌─────────────────────────────────────────┐
│         Microservices (via RPC)          │
└─────────────────────────────────────────┘
```

## Integration

### In `micro run` (Development)

```go
// cmd/micro/run/run.go
import "go-micro.dev/v5/gateway/api"

gw, err := api.New(api.Options{
    Address:     ":8080",
    AuthEnabled: false, // No auth in dev mode
    HandlerRegistrar: func(mux *http.ServeMux) error {
        // Register dev-mode handlers (no auth)
        mux.HandleFunc("/", dashboardHandler)
        mux.HandleFunc("/api/", apiHandler)
        return nil
    },
})
```

### In `micro server` (Production)

```go
// cmd/micro/server/server.go
import "go-micro.dev/v5/gateway/api"

gw, err := api.New(api.Options{
    Address:     ":8080",
    AuthEnabled: true, // Auth required in production
    HandlerRegistrar: func(mux *http.ServeMux) error {
        // Register prod handlers with auth middleware
        mux.HandleFunc("/", authMiddleware(dashboardHandler))
        mux.HandleFunc("/api/", authMiddleware(apiHandler))
        return nil
    },
})
```

### Custom Application

```go
// Your app
import "go-micro.dev/v5/gateway/api"

func main() {
    gw, err := api.New(api.Options{
        Address: ":8080",
        HandlerRegistrar: func(mux *http.ServeMux) error {
            // Your custom handlers
            mux.HandleFunc("/health", healthHandler)
            mux.HandleFunc("/metrics", metricsHandler)
            mux.HandleFunc("/api/", proxyToServices)
            return nil
        },
    })

    if err != nil {
        log.Fatal(err)
    }

    log.Println("Gateway running on :8080")
    gw.Wait()
}
```

## Comparison with Old Architecture

### Before (Duplicated Code)

```
cmd/micro/run/gateway/
  └── gateway.go (300+ lines)

cmd/micro/server/
  └── gateway.go (150+ lines)

❌ Code duplication
❌ Inconsistent behavior
❌ Hard to reuse
```

### After (Unified)

```
gateway/api/
  └── gateway.go (150 lines, reusable)

cmd/micro/server/
  └── gateway.go (70 lines, compatibility wrapper)

cmd/micro/run/
  └── Uses api.New() directly

✅ Single source of truth
✅ Consistent behavior
✅ Easy to reuse in custom apps
```

## Benefits

1. **Reusability** - Use in any Go application, not just micro CLI
2. **Testability** - Easy to test with custom handler registrars
3. **Flexibility** - Supports different configurations (dev, prod, custom)
4. **Consistency** - Same gateway code for all use cases
5. **Maintainability** - One place to fix bugs and add features

## Migration Guide

### From `cmd/micro/server/gateway.go`

**Before:**
```go
import "go-micro.dev/v5/cmd/micro/server"

gw, err := server.StartGateway(server.GatewayOptions{
    Address: ":8080",
    AuthEnabled: true,
    Store: myStore,
})
```

**After:**
```go
import "go-micro.dev/v5/gateway/api"

gw, err := api.New(api.Options{
    Address: ":8080",
    AuthEnabled: true,
    HandlerRegistrar: func(mux *http.ServeMux) error {
        // Register your handlers
        // Pass store as closure
        return registerHandlers(mux, myStore)
    },
})
```

## Examples

See:
- `cmd/micro/server/gateway.go` - Production gateway with auth
- `cmd/micro/run/run.go` - Development gateway without auth
- `examples/gateway/` - Custom gateway examples (coming soon)

## License

Apache 2.0
