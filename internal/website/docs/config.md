---
layout: default
---

# Configuration

Go Micro follows a progressive configuration model so you can start with zero setup and layer in complexity only when needed.

## Levels of Configuration

1. Zero Config (Defaults)
   - mDNS registry, HTTP transport, in-memory broker/store
2. Environment Variables
   - Override core components without code changes
3. Code Options
   - Fine-grained control via functional options
4. External Sources (Future / Plugins)
   - Configuration loaded from files, vaults, or remote services

## Core Environment Variables

| Component | Variable | Example | Purpose |
|-----------|----------|---------|---------|
| Registry  | `MICRO_REGISTRY` | `MICRO_REGISTRY=consul` | Select registry implementation |
| Registry Address | `MICRO_REGISTRY_ADDRESS` | `MICRO_REGISTRY_ADDRESS=127.0.0.1:8500` | Point to registry service |
| Broker    | `MICRO_BROKER` | `MICRO_BROKER=nats` | Select broker implementation |
| Broker Address | `MICRO_BROKER_ADDRESS` | `MICRO_BROKER_ADDRESS=nats://localhost:4222` | Broker endpoint |
| Transport | `MICRO_TRANSPORT` | `MICRO_TRANSPORT=nats` | Select transport implementation |
| Transport Address | `MICRO_TRANSPORT_ADDRESS` | `MICRO_TRANSPORT_ADDRESS=nats://localhost:4222` | Transport endpoint |
| Store     | `MICRO_STORE` | `MICRO_STORE=postgres` | Select store implementation |
| Store Database | `MICRO_STORE_DATABASE` | `MICRO_STORE_DATABASE=app` | Logical database name |
| Store Table | `MICRO_STORE_TABLE` | `MICRO_STORE_TABLE=records` | Default table/collection |
| Store Address | `MICRO_STORE_ADDRESS` | `MICRO_STORE_ADDRESS=postgres://user:pass@localhost:5432/app?sslmode=disable` | Connection string |
| Server Address | `MICRO_SERVER_ADDRESS` | `MICRO_SERVER_ADDRESS=:8080` | Bind address for RPC server |

## Example: Switching Components via Env Vars

```bash
# Use NATS for broker and transport, Consul for registry
export MICRO_BROKER=nats
export MICRO_TRANSPORT=nats
export MICRO_REGISTRY=consul
export MICRO_REGISTRY_ADDRESS=127.0.0.1:8500

# Run your service
go run main.go
```

No code changes required. The framework internally wires the selected implementations.

## Equivalent Code Configuration

```go
service := micro.NewService(
    micro.Name("helloworld"),
    micro.Broker(nats.NewBroker()),
    micro.Transport(natstransport.NewTransport()),
    micro.Registry(consul.NewRegistry(registry.Addrs("127.0.0.1:8500"))),
)
service.Init()
```

Use env vars for deployment level overrides; use code options for explicit control or when composing advanced setups.

## Precedence Rules

1. Explicit code options always win
2. If not set in code, env vars are applied
3. If neither code nor env vars set, defaults are used

## Discoverability Strategy

Defaults allow local development with zero friction. As teams scale:
- Introduce env vars for staging/production parity
- Consolidate secrets (e.g. store passwords) using external secret managers (future guide)
- Move to service mesh aware registry (Consul/NATS JetStream)

## Validating Configuration

Enable debug logging to confirm selected components:

```bash
MICRO_LOG_LEVEL=debug go run main.go
```

You will see lines like:

```text
Registry [consul] Initialised
Broker [nats] Connected
Transport [nats] Listening on nats://localhost:4222
Store [postgres] Connected to app/records
```

## Patterns

### Twelve-Factor Alignment
Environment variables map directly to deploy-time configuration. Avoid hardcoding component choices so services remain portable.

### Multi-Environment Setup
Use a simple env file per environment:

```bash
# .env.staging
MICRO_REGISTRY=consul
MICRO_REGISTRY_ADDRESS=consul.staging.internal:8500
MICRO_BROKER=nats
MICRO_BROKER_ADDRESS=nats.staging.internal:4222
MICRO_STORE=postgres
MICRO_STORE_ADDRESS=postgres://staging:pass@pg.staging.internal:5432/app?sslmode=disable
```

Load with your process manager or container orchestrator.

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Service starts with memory store unexpectedly | Env vars not exported | `env | grep MICRO_STORE` to verify |
| Consul errors about connection refused | Wrong address/port | Check `MICRO_REGISTRY_ADDRESS` value |
| NATS connection timeout | Server not running | Start NATS or change address |
| Postgres SSL errors | Missing sslmode param | Append `?sslmode=disable` locally |

## Related

- [ADR-009: Progressive Configuration](architecture/adr-009-progressive-configuration.md)
- [Getting Started](getting-started.md)
- [Plugins](plugins.md)
