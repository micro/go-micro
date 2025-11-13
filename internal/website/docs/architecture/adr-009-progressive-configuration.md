---
layout: default
---

# ADR-009: Progressive Configuration

## Status
**Accepted**

## Context

Microservices frameworks face a paradox:
- Beginners want "Hello World" to work immediately
- Production needs sophisticated configuration

Too simple: Framework is toy, not production-ready
Too complex: High barrier to entry, discourages adoption

## Decision

Implement **progressive configuration** where:

1. **Zero config** works for development
2. **Environment variables** provide simple overrides
3. **Code-based options** enable fine-grained control
4. **Defaults are production-aware** but not production-ready

## Levels of Configuration

### Level 1: Zero Config (Development)
```go
svc := micro.NewService(micro.Name("hello"))
svc.Run()
```

Uses defaults:
- mDNS registry (local)
- HTTP transport
- Random available port
- Memory broker/store

### Level 2: Environment Variables (Staging)
```bash
MICRO_REGISTRY=consul \
MICRO_REGISTRY_ADDRESS=consul:8500 \
MICRO_BROKER=nats \
MICRO_BROKER_ADDRESS=nats://nats:4222 \
./service
```

No code changes, works with CLI flags.

### Level 3: Code Options (Production)
```go
reg := consul.NewConsulRegistry(
    registry.Addrs("consul1:8500", "consul2:8500"),
    registry.TLSConfig(tlsConf),
)

b := nats.NewNatsBroker(
    broker.Addrs("nats://nats1:4222", "nats://nats2:4222"),
    nats.DrainConnection(),
)

svc := micro.NewService(
    micro.Name("myservice"),
    micro.Version("1.2.3"),
    micro.Registry(reg),
    micro.Broker(b),
    micro.Address(":8080"),
)
```

Full control over initialization and configuration.

### Level 4: External Config (Enterprise)
```go
cfg := config.NewConfig(
    config.Source(file.NewSource("config.yaml")),
    config.Source(env.NewSource()),
    config.Source(vault.NewSource()),
)

// Use cfg to initialize plugins with complex configs
```

## Environment Variable Patterns

Standard vars for all plugins:
```bash
MICRO_REGISTRY=<type>              # consul, etcd, nats, mdns
MICRO_REGISTRY_ADDRESS=<addrs>     # Comma-separated
MICRO_BROKER=<type>
MICRO_BROKER_ADDRESS=<addrs>
MICRO_TRANSPORT=<type>
MICRO_TRANSPORT_ADDRESS=<addrs>
MICRO_STORE=<type>
MICRO_STORE_ADDRESS=<addrs>
MICRO_STORE_DATABASE=<name>
MICRO_STORE_TABLE=<name>
```

Plugin-specific vars:
```bash
ETCD_USERNAME=user
ETCD_PASSWORD=pass
CONSUL_TOKEN=secret
```

## Consequences

### Positive

- **Fast start**: Beginners productive immediately
- **Easy deployment**: Env vars for different environments
- **Power when needed**: Full programmatic control available
- **Learn incrementally**: Complexity introduced as required

### Negative

- **Three config sources**: Environment, code, and CLI flags can conflict
- **Documentation**: Must explain all levels clearly
- **Testing**: Need to test all configuration methods

### Mitigations

- Clear precedence: Code options > Environment > Defaults
- Comprehensive examples for each level
- Validation and helpful error messages

## Validation Example

```go
func (s *service) Init() error {
    if s.opts.Name == "" {
        return errors.New("service name required")
    }
    
    // Warn about development defaults in production
    if isProduction() && usingDefaults() {
        log.Warn("Using development defaults in production")
    }
    
    return nil
}
```

## Related

- [ADR-004: mDNS as Default Registry](adr-004-mdns-default-registry.md)
- ADR-008: Environment Variable Support (planned)
- [Getting Started Guide](../getting-started.md) - Configuration examples
 - [Configuration Guide](../config.md)
