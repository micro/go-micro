---
layout: default
---

# micro run - Local Development

`micro run` provides a Rails/Spring-like development experience for Go microservices.

## Quick Start

```bash
# Run services in current directory with hot reload
micro run

# Run from a specific directory
micro run ./myapp

# Clone and run from GitHub
micro run github.com/micro/blog
```

## Features

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

## Tips

1. **Port Configuration**: Set `port` for services that expose HTTP to enable health check waiting
2. **Health Endpoint**: Implement `/health` returning 200 for reliable startup sequencing
3. **Environment Separation**: Keep secrets in production env, use file:// paths for development
4. **Hot Reload Scope**: Only `.go` files trigger rebuilds; static assets don't
