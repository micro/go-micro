# Web Service Example

HTTP web service with automatic service discovery and registration.

## What It Does

This example creates an HTTP service that:
- Serves RESTful API endpoints
- Registers with service discovery
- Provides health checks
- Uses standard Go HTTP handlers

## Run It

```bash
go run main.go
```

## Test It

```bash
# Get service info
curl http://localhost:9090/

# List all users
curl http://localhost:9090/users

# Get specific user
curl http://localhost:9090/users/1

# Health check
curl http://localhost:9090/health
```

## Key Features

- **Standard HTTP**: Use familiar `http.Handler` interface
- **Service Discovery**: Automatically registers with registry
- **Health Checks**: Built-in health endpoint
- **JSON APIs**: Easy REST API development

## When to Use

Use `web.Service` when:
- Building REST APIs
- Serving web UIs
- Working with HTTP-specific features
- Migrating existing HTTP services

Use regular `micro.Service` when:
- Building RPC services
- Need bidirectional streaming
- Want automatic load balancing
- Prefer structured RPC over HTTP

## Next Steps

- See [hello-world](../hello-world/) for RPC services
- See [production-ready](../production-ready/) for observability
