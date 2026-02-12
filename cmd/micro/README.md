# Micro

Go Micro Command Line

## Install the CLI

Install `micro` via `go install`

```
go install go-micro.dev/v5/cmd/micro@v5.16.0
```


## Create a service

Create your service (all setup is now automatic!):

```
micro new helloworld
```

This will:
- Create a new service in the `helloworld` directory
- Automatically run `go mod tidy` and `make proto` for you
- Show the updated project tree including generated files
- Warn you if `protoc` is not installed, with install instructions

## Run the service

Run your service:

```
micro run
```

This starts:
- **API Gateway** on http://localhost:8080
- **Web Dashboard** at http://localhost:8080
- **Agent Playground** at http://localhost:8080/agent
- **API Explorer** at http://localhost:8080/api
- **MCP Tools** at http://localhost:8080/api/mcp/tools
- **Hot Reload** watching for file changes
- **Services** in dependency order

Open http://localhost:8080 to see your services and call them from the browser.

### Output

```
  ┌─────────────────────────────────────────────────────────────┐
  │                                                             │
  │   Micro                                                     │
  │                                                             │
  │   Web:     http://localhost:8080                            │
  │   API:     http://localhost:8080/api/{service}/{method}     │
  │   Health:  http://localhost:8080/health                     │
  │                                                             │
  │   Services:                                                 │
  │     ● helloworld                                            │
  │                                                             │
  │   Watching for changes...                                   │
  │                                                             │
  └─────────────────────────────────────────────────────────────┘
```

### Options

```
micro run                    # Gateway on :8080, hot reload enabled
micro run --address :3000    # Gateway on custom port
micro run --no-gateway       # Services only, no HTTP gateway
micro run --no-watch         # Disable hot reload
micro run --env production   # Use production environment
micro run github.com/micro/blog  # Clone and run from GitHub
```

### Calling Services

Via curl:
```bash
curl -X POST http://localhost:8080/api/helloworld/Helloworld.Call -d '{"name": "World"}'
```

Or browse to http://localhost:8080 and use the web interface.

List services:
```
micro services
```

## Configuration (micro.mu)

For multi-service projects, create a `micro.mu` file to define services, dependencies, and environments:

```
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

env development
    STORE_ADDRESS file://./data
    DEBUG true

env production
    STORE_ADDRESS postgres://localhost/db
```

### Configuration Options

| Property | Description |
|----------|-------------|
| `path` | Directory containing the service (with main.go) |
| `port` | Port the service listens on (for health checks) |
| `depends` | Services that must start first (space-separated) |

### Environment Management

Environment variables are injected based on the `--env` flag:

```
micro run                    # Uses 'development' env (default)
micro run --env production   # Uses 'production' env
MICRO_ENV=staging micro run  # Uses 'staging' env
```

### JSON Alternative

You can also use `micro.json` if you prefer:

```json
{
  "services": {
    "users": { "path": "./users", "port": 8081 },
    "posts": { "path": "./posts", "port": 8082, "depends": ["users"] }
  },
  "env": {
    "development": { "STORE_ADDRESS": "file://./data" }
  }
}
```

### Without Configuration

If no `micro.mu` or `micro.json` exists, `micro run` discovers all `main.go` files and runs them (original behavior).

## Describe the service

Describe the service to see available endpoints

```
micro describe helloworld
```

Output

```
{
    "name": "helloworld",
    "version": "latest",
    "metadata": null,
    "endpoints": [
        {
            "request": {
                "name": "Request",
                "type": "Request",
                "values": [
                    {
                        "name": "name",
                        "type": "string",
                        "values": null
                    }
                ]
            },
            "response": {
                "name": "Response",
                "type": "Response",
                "values": [
                    {
                        "name": "msg",
                        "type": "string",
                        "values": null
                    }
                ]
            },
            "metadata": {},
            "name": "Helloworld.Call"
        },
        {
            "request": {
                "name": "Context",
                "type": "Context",
                "values": null
            },
            "response": {
                "name": "Stream",
                "type": "Stream",
                "values": null
            },
            "metadata": {
                "stream": "true"
            },
            "name": "Helloworld.Stream"
        }
    ],
    "nodes": [
        {
            "metadata": {
                "broker": "http",
                "protocol": "mucp",
                "registry": "mdns",
                "server": "mucp",
                "transport": "http"
            },
            "id": "helloworld-31e55be7-ac83-4810-89c8-a6192fb3ae83",
            "address": "127.0.0.1:39963"
        }
    ]
}
```

## Call the service

Call via RPC endpoint

```
micro call helloworld Helloworld.Call '{"name": "Asim"}'
```

## Create a client

Create a client to call the service

```go
package main

import (
        "context"
        "fmt"

        "go-micro.dev/v5"
)

type Request struct {
        Name string
}

type Response struct {
        Message string
}

func main() {
        client := micro.New("helloworld").Client()

        req := client.NewRequest("helloworld", "Helloworld.Call", &Request{Name: "John"})

        var rsp Response

        err := client.Call(context.TODO(), req, &rsp)
        if err != nil {
                fmt.Println(err)
                return
        }

        fmt.Println(rsp.Message)
}
```

## Building and Deployment

### Build Binaries

Build Go binaries for deployment:

```bash
micro build                     # Build for current OS
micro build --os linux          # Cross-compile for Linux
micro build --os linux --arch arm64  # For ARM64
micro build --output ./dist     # Custom output directory
```

### Deploy to Server

Deploy to any Linux server with systemd:

```bash
# First time: set up the server
ssh user@server
curl -fsSL https://go-micro.dev/install.sh | sh
sudo micro init --server
exit

# Deploy from your laptop
micro deploy user@server
```

The deploy command:
1. Builds binaries for linux/amd64
2. Copies via SSH to `/opt/micro/bin/`
3. Sets up systemd services (`micro@<service>`)
4. Restarts and verifies services are running

### Named Deploy Targets

Add deploy targets to `micro.mu`:

```
deploy prod
    ssh deploy@prod.example.com

deploy staging
    ssh deploy@staging.example.com
```

Then:
```bash
micro deploy prod      # Deploy to production
micro deploy staging   # Deploy to staging
```

### Managing Deployed Services

```bash
# Check status
micro status --remote user@server

# View logs
micro logs --remote user@server
micro logs myservice --remote user@server -f

# Stop a service
micro stop myservice --remote user@server
```

See [internal/website/docs/deployment.md](../../internal/website/docs/deployment.md) for the full deployment guide.

## Protobuf 

Use protobuf for code generation with [protoc-gen-micro](https://github.com/micro/go-micro/tree/master/cmd/protoc-gen-micro)

## Server

The micro server is a production web dashboard and authenticated API gateway for interacting with services that are already running (e.g., managed by systemd via `micro deploy`). It does **not** build, run, or watch services — for local development, use `micro run` instead.

Run it like so

```
micro server
```

Then browse to [localhost:8080](http://localhost:8080) and log in with the default admin account (`admin`/`micro`).

### API Endpoints 

The API provides a fixed HTTP entrypoint for calling services

```
curl http://localhost:8080/api/helloworld/Helloworld/Call -d '{"name": "John"}'
```
See /api for more details and documentation for each service

### Web Dashboard 

The web dashboard provides a modern, secure UI for managing and exploring your Micro services. Major features include:

- **Dynamic Service & Endpoint Forms**: Browse all registered services and endpoints. For each endpoint, a dynamic form is generated for easy testing and exploration.
- **API Documentation**: The `/api` page lists all available services and endpoints, with request/response schemas and a sidebar for quick navigation. A documentation banner explains authentication requirements.
- **JWT Authentication**: All login and token management uses a custom JWT utility. Passwords are securely stored with bcrypt. All `/api/x` endpoints and authenticated pages require an `Authorization: Bearer <token>` header (or `micro_token` cookie as fallback).
- **Token Management**: The `/auth/tokens` page allows you to generate, view (obfuscated), and copy JWT tokens. Tokens are stored and can be revoked. When a user is deleted, all their tokens are revoked immediately.
- **User Management**: The `/auth/users` page allows you to create, list, and delete users. Passwords are never shown or stored in plaintext.
- **Token Revocation**: JWT tokens are stored and checked for revocation on every request. Revoked or deleted tokens are immediately invalidated.
- **Security**: All protected endpoints use consistent authentication logic. Unauthorized or revoked tokens receive a 401 error. All sensitive actions require authentication.
- **Logs & Status**: View service logs and status (PID, uptime, etc) directly from the dashboard.

To get started, run:

```
micro server
```

Then browse to [localhost:8080](http://localhost:8080) and log in with the default admin account (`admin`/`micro`).

> **Note:** See the `/api` page for details on API authentication and how to generate tokens for use with the HTTP API

## Gateway Architecture

The `micro run` and `micro server` commands both use a unified gateway implementation (`cmd/micro/server/gateway.go`), providing consistent HTTP-to-RPC translation, service discovery, and web UI capabilities.

### Key Differences

| Feature | `micro run` | `micro server` |
|---------|-------------|----------------|
| **Purpose** | Development | Production |
| **Authentication** | Enabled (default `admin`/`micro`) | Enabled (default `admin`/`micro`) |
| **Process Management** | Yes (builds/runs services) | No (assumes services running) |
| **Hot Reload** | Yes (watches files) | No |
| **Scopes** | Available (`/auth/scopes`) | Available (`/auth/scopes`) |
| **Use Case** | Local development | Deployed API gateway |

### Why Unified?

Previously, each command had its own gateway implementation, leading to code duplication. The unified gateway means:

- New features (like MCP integration) benefit both commands
- Consistent behavior between development and production
- Single codebase to test and maintain
- Same HTTP API, web UI, and service discovery logic

### Gateway Features

Both commands provide:

- **HTTP API**: `POST /api/{service}/{endpoint}` with JSON request/response
- **Service Discovery**: Automatic detection via registry (mdns/consul/etcd)
- **Health Checks**: `/health`, `/health/live`, `/health/ready` endpoints
- **Web Dashboard**: Browse services, test endpoints, view documentation
- **Hot Service Updates**: Gateway automatically picks up new service registrations
- **JWT Authentication**: Tokens, user management, login at `/auth/login`, `/auth/tokens`, `/auth/users`
- **Endpoint Scopes**: Restrict which tokens can call which endpoints via `/auth/scopes`
- **MCP Integration**: AI tools at `/api/mcp/tools`, agent playground at `/agent`

### Authentication & Scopes

Both `micro run` and `micro server` use the same `auth.Account` type from the go-micro framework. The gateway stores accounts under `auth/<id>` in the default store and uses JWT tokens with RSA256 signing.

**Scope enforcement** applies to all call paths:

| Path | Description |
|------|-------------|
| `POST /api/{service}/{endpoint}` | HTTP API calls |
| `POST /api/mcp/call` | MCP tool invocations |
| Agent playground | Tool calls made by the AI agent |

Scopes are configured via the web UI at `/auth/scopes`. Each endpoint can require one or more scopes. A token must carry at least one matching scope to call a protected endpoint. The `*` scope on a token bypasses all checks. Endpoints with no scopes set are open to any authenticated token.

See the [Scopes](#scopes) section below for details.

### Development Mode (`micro run`)

```bash
micro run  # Auth enabled, default admin/micro
```

- Authentication enabled with default credentials (`admin`/`micro`)
- Web UI requires login
- Scopes available for testing access control
- Ideal for development with realistic auth behavior

### Production Mode (`micro server`)

```bash
micro server  # Auth enabled, JWT tokens required
```

- JWT authentication on all API calls
- User/token management via web UI
- Secure by default
- Login required: default credentials `admin/micro`

### Programmatic Gateway Usage

You can also start the gateway programmatically in your own Go code:

```go
import "go-micro.dev/v5/cmd/micro/server"

// Start gateway with auth (recommended)
gw, err := server.StartGateway(server.GatewayOptions{
    Address:     ":8080",
    AuthEnabled: true,
})

// Start gateway without auth (testing only)
gw, err := server.StartGateway(server.GatewayOptions{
    Address:     ":8080",
    AuthEnabled: false,
})
```

See [`internal/website/docs/architecture/adr-010-unified-gateway.md`](../../internal/website/docs/architecture/adr-010-unified-gateway.md) for architecture details.

### Scopes

Scopes provide fine-grained access control over which tokens can call which service endpoints. They are managed through the web UI at `/auth/scopes` and enforced on every call through the gateway.

#### How It Works

1. **Define scopes on endpoints** — Visit `/auth/scopes` and set required scopes for each service endpoint (e.g., set `billing` on `payments.Payments.Charge`)
2. **Create tokens with scopes** — Visit `/auth/tokens` and create tokens with matching scopes (e.g., a token with `billing` scope)
3. **Scopes are enforced** — When a token calls an endpoint, the gateway checks that the token has at least one scope matching the endpoint's required scopes

#### Scope Matching Rules

- Scopes are **exact string matches** — `billing` on a token matches `billing` on an endpoint
- A token with `*` scope bypasses all scope checks (admin wildcard)
- Endpoints with **no scopes set** are open to any valid token
- An endpoint can require **multiple scopes** — the token needs to match just one
- Scope names are free-form strings — use whatever convention fits your project

#### Common Patterns

| Pattern | Endpoint Scopes | Token Scopes | Result |
|---------|----------------|--------------|--------|
| Protect a service | Set `greeter` on all greeter endpoints (use Bulk Set with `greeter.*`) | Token with `greeter` | Token can call any greeter endpoint |
| Restrict an endpoint | Set `billing` on `payments.Payments.Charge` | Token with `billing` | Only that endpoint is restricted |
| Role-based | Set `admin` on sensitive endpoints | Admin token with `admin`, user token with `user` | Only admin tokens can call sensitive endpoints |
| Full access | Any | Token with `*` | Bypasses all scope checks |

#### Relationship to Framework Auth

The gateway's scope system uses `auth.Account` from the go-micro framework. Scopes on accounts are the same `[]string` field used by the framework's `auth.Rules` and `wrapper/auth` package. The gateway stores scope requirements in the default store under `endpoint-scopes/<service>.<endpoint>` keys and checks them on every HTTP request.

For service-level (RPC) auth within the go-micro mesh, use the `wrapper/auth` package which provides `auth.Rules` with priority-based access control. See the [auth wrapper documentation](../../wrapper/auth/README.md) for details.
