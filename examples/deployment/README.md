# Docker Compose Deployment Example

Run a go-micro service with MCP gateway, service registry, and distributed tracing in one command.

## Architecture

```
┌─────────┐     discover     ┌──────────┐     RPC      ┌─────────┐
│  Agent   │ ─────────────→  │   MCP    │ ──────────→  │  Your   │
│ (Claude) │    MCP :3001    │ Gateway  │              │ Service │
└─────────┘                  └──────────┘              └─────────┘
                                  │                        │
                                  ▼                        ▼
                             ┌──────────┐           ┌──────────┐
                             │  Consul  │           │  Jaeger  │
                             │ Registry │           │ Tracing  │
                             │   :8500  │           │  :16686  │
                             └──────────┘           └──────────┘
```

## Quick Start

```bash
docker-compose up
```

## Endpoints

| Service | URL |
|---------|-----|
| MCP Tools | http://localhost:3001/mcp/tools |
| Consul UI | http://localhost:8500 |
| Jaeger UI | http://localhost:16686 |
| Service RPC | http://localhost:9090 |

## Test

```bash
# List MCP tools
curl http://localhost:3001/mcp/tools | jq

# Call a tool
curl -X POST http://localhost:3001/mcp/call \
  -H 'Content-Type: application/json' \
  -d '{"tool": "myservice.Handler.Method", "arguments": {"key": "value"}}'

# View traces in Jaeger
open http://localhost:16686
```

## Connect Claude Code

```bash
# Claude Code can connect to the running MCP gateway
# Add to your Claude Code MCP settings:
```

```json
{
  "mcpServers": {
    "my-services": {
      "url": "http://localhost:3001/mcp"
    }
  }
}
```

## Customizing

### Add Your Service

Replace the `app` service's build context with your service directory:

```yaml
app:
  build:
    context: ../path/to/your/service
    dockerfile: Dockerfile
```

### Add More Services

```yaml
users:
  build: ./users
  environment:
    MICRO_REGISTRY: consul
    MICRO_REGISTRY_ADDRESS: consul:8500

orders:
  build: ./orders
  environment:
    MICRO_REGISTRY: consul
    MICRO_REGISTRY_ADDRESS: consul:8500
```

All services register with Consul. The MCP gateway discovers them automatically.

### Add Redis Cache

```yaml
redis:
  image: redis:7-alpine
  ports:
    - "6379:6379"
```

Then set `MICRO_CACHE_ADDRESS=redis:6379` on your service.

### Production Considerations

- Add health checks to each service
- Use named volumes for Consul data persistence
- Configure rate limiting on the MCP gateway
- Set up TLS between services
- Use secrets management for API keys
