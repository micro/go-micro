---
layout: default
---

# micro run - Local Development

`micro run` provides a complete development environment for Go microservices.

> **Note**: This guide focuses on `micro run` features. For a comparison with `micro server` and gateway architecture details, see the [CLI & Gateway Guide](cli-gateway.md).

## Quick Start

```bash
micro new helloworld
cd helloworld
micro run
```

Open http://localhost:8080 to see your service.

## What You Get

When you run `micro run`, you get:

| URL | Description |
|-----|-------------|
| http://localhost:8080 | Web dashboard - browse and call services |
| http://localhost:8080/api/{service}/{method} | API gateway - HTTP to RPC proxy |
| http://localhost:8080/health | Health checks - aggregated service health |
| http://localhost:8080/services | Service list - JSON |

Plus:
- **Hot Reload** - File changes trigger automatic rebuild
- **Dependency Ordering** - Services start in the right order
- **Environment Management** - Dev/staging/production configs

## Features

### API Gateway

The gateway converts HTTP requests to RPC calls:

```bash
# Call a service method
curl -X POST http://localhost:8080/api/helloworld/Say.Hello \
  -d '{"name": "World"}'

# Response
{"message": "Hello World"}
```

### Hot Reload

By default, `micro run` watches for `.go` file changes and automatically rebuilds and restarts affected services.

```bash
micro run              # Hot reload enabled (default)
micro run --no-watch   # Disable hot reload
```

Changes are debounced (300ms) to handle rapid saves from editors.

### Configuration File

For multi-service projects, create a `micro.mu` file to define services, dependencies, and environments.

#### micro.mu (Recommended)

```
# Service definitions
service users
    path ./users
    port 8081

service posts
    path ./posts
    port 8082
    depends users

service web
    path ./web
    port 8089
    depends users posts

# Environment configurations
env development
    STORE_ADDRESS file://./data
    DEBUG true

env production
    STORE_ADDRESS postgres://localhost/db
    DEBUG false
```

#### micro.json (Alternative)

```json
{
  "services": {
    "users": {
      "path": "./users",
      "port": 8081
    },
    "posts": {
      "path": "./posts",
      "port": 8082,
      "depends": ["users"]
    }
  },
  "env": {
    "development": {
      "STORE_ADDRESS": "file://./data"
    }
  }
}
```

### Service Properties

| Property | Required | Description |
|----------|----------|-------------|
| `path` | Yes | Directory containing the service (with main.go) |
| `port` | No | Port the service listens on (enables health check waiting) |
| `depends` | No | Services that must start first (space-separated in .mu, array in .json) |

### Dependency Ordering

When `depends` is specified, services start in topological order:

1. Services with no dependencies start first
2. Each service waits for its dependencies to be ready
3. If a service has a `port`, we wait for `/health` to return 200
4. Circular dependencies are detected and reported as errors

### Environment Management

```bash
micro run                    # Uses 'development' (default)
micro run --env production   # Uses 'production'
micro run --env staging      # Uses 'staging'
MICRO_ENV=test micro run     # Environment variable override
```

Environment variables from the config are injected into each service's environment.

### Graceful Shutdown

On SIGINT (Ctrl+C) or SIGTERM:

1. Services stop in reverse dependency order
2. SIGTERM is sent first (graceful)
3. After 5 seconds, SIGKILL if still running
4. PID files are cleaned up

## Without Configuration

If no `micro.mu` or `micro.json` exists:

1. All `main.go` files are discovered recursively
2. Each is built and run
3. No dependency ordering
4. Hot reload still works

## Logs

Service logs are written to:
- Terminal: Colorized with service name prefix
- File: `~/micro/logs/{service}-{hash}.log`

View logs:
```bash
micro logs          # List available logs
micro logs users    # Show logs for 'users' service
```

## Process Management

```bash
micro status        # Show running services
micro stop users    # Stop a specific service
```

## Example: micro/blog

The [micro/blog](https://github.com/micro/blog) project demonstrates a multi-service setup:

```
# micro.mu
service users
    path ./users
    port 8081

service posts
    path ./posts
    port 8082
    depends users

service comments
    path ./comments
    port 8083
    depends users posts

service web
    path ./web
    port 8089
    depends users posts comments
```

Run it:
```bash
micro run github.com/micro/blog
```

## Options

```bash
micro run                    # Gateway on :8080, hot reload
micro run --address :3000    # Custom gateway port
micro run --no-gateway       # Services only, no HTTP gateway
micro run --no-watch         # Disable hot reload
micro run --env production   # Use production environment
```

## Tips

1. **Browse First**: Open http://localhost:8080 to explore your services
2. **Port Configuration**: Set `port` for services to enable health check waiting
3. **Health Endpoint**: Implement `/health` returning 200 for reliable startup sequencing
4. **Environment Separation**: Keep secrets in production env, use file:// paths for development
5. **Hot Reload Scope**: Only `.go` files trigger rebuilds; static assets don't
