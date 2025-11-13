---
layout: default
---

# ADR-004: mDNS as Default Registry

## Status
**Accepted**

## Context

Service discovery is critical for microservices. Common approaches:

1. **Central registry** (Consul, Etcd) - Requires infrastructure
2. **DNS-based** (Kubernetes DNS) - Platform-specific
3. **Static configuration** - Doesn't scale
4. **Multicast DNS (mDNS)** - Zero-config, local network

For local development and getting started, requiring infrastructure setup is a barrier. Production deployments typically have existing service discovery infrastructure.

## Decision

Use **mDNS as the default registry** for service discovery.

- Works immediately on local networks
- No external dependencies
- Suitable for development and simple deployments
- Easily swapped for production registries (Consul, Etcd, Kubernetes)

## Implementation

```go
// Default - uses mDNS automatically
svc := micro.NewService(micro.Name("myservice"))

// Production - swap to Consul
reg := consul.NewConsulRegistry()
svc := micro.NewService(
    micro.Name("myservice"),
    micro.Registry(reg),
)
```

## Consequences

### Positive

- **Zero setup**: `go run main.go` just works
- **Fast iteration**: No infrastructure for local dev
- **Learning curve**: Newcomers start immediately
- **Progressive complexity**: Add infrastructure as needed

### Negative

- **Local network only**: mDNS doesn't cross subnets/VLANs
- **Not for production**: Needs proper registry in production
- **Port 5353**: May conflict with existing mDNS services
- **Discovery delay**: Can take 1-2 seconds

### Mitigations

- Clear documentation on production alternatives
- Environment variables for easy swapping (`MICRO_REGISTRY=consul`)
- Examples for all major registries
- Health checks and readiness probes for production

## Use Cases

### Good for mDNS
- Local development
- Testing
- Simple internal services on same network
- Learning and prototyping

### Need Production Registry
- Cross-datacenter communication
- Cloud deployments
- Large service mesh (100+ services)
- Require advanced features (health checks, metadata filtering)

## Alternatives Considered

### No Default (Force Configuration)
Rejected because:
- Poor first-run experience
- Increases barrier to entry
- Users must setup infrastructure before trying framework

### Static Configuration
Rejected because:
- Doesn't support dynamic service discovery
- Manual configuration doesn't scale
- Doesn't reflect real microservices usage

### Consul as Default
Rejected because:
- Requires running Consul for "Hello World"
- Platform-specific
- Adds complexity for beginners

## Migration Path

Start with mDNS, migrate to production registry:

```bash
# Development
go run main.go

# Staging
MICRO_REGISTRY=consul MICRO_REGISTRY_ADDRESS=consul:8500 go run main.go

# Production (Kubernetes)
MICRO_REGISTRY=nats MICRO_REGISTRY_ADDRESS=nats://nats:4222 ./service
```

## Related

- [ADR-001: Plugin Architecture](adr-001-plugin-architecture.md)
- [ADR-009: Progressive Configuration](adr-009-progressive-configuration.md)
- [Registry Documentation](../registry.md)
