---
layout: default
---

# ADR-001: Plugin Architecture

## Status
**Accepted**

## Context

Microservices frameworks need to support multiple infrastructure backends (registries, brokers, transports, stores). Different teams have different preferences and existing infrastructure.

Hard-coding specific implementations:
- Limits framework adoption
- Forces migration of existing infrastructure
- Prevents innovation and experimentation

## Decision

Go Micro uses a **pluggable architecture** where:

1. Core interfaces define contracts (Registry, Broker, Transport, Store, etc.)
2. Multiple implementations live in the same repository under interface directories
3. Plugins are imported directly and passed via options
4. Default implementations work without any infrastructure

## Structure

```
go-micro/
├── registry/          # Interface definition
│   ├── registry.go
│   ├── mdns.go       # Default implementation
│   ├── consul/       # Plugin
│   ├── etcd/         # Plugin
│   └── nats/         # Plugin
├── broker/
├── transport/
└── store/
```

## Consequences

### Positive

- **No version hell**: Plugins versioned with core framework
- **Discovery**: Users browse available plugins in same repo
- **Consistency**: All plugins follow same patterns
- **Testing**: Plugins tested together
- **Zero config**: Default implementations require no setup

### Negative

- **Repo size**: More code in one repository
- **Plugin maintenance**: Core team responsible for plugin quality
- **Breaking changes**: Harder to evolve individual plugins independently

### Neutral

- Plugins can be extracted to separate repos if they grow complex
- Community can contribute plugins via PR
- Plugin-specific issues easier to triage

## Alternatives Considered

### Separate Plugin Repositories
Used by go-kit and other frameworks. Rejected because:
- Version compatibility becomes user's problem
- Discovery requires documentation
- Testing integration harder
- Splitting community

### Single Implementation
Like standard `net/http`. Rejected because:
- Forces infrastructure choices
- Limits adoption
- Can't leverage existing infrastructure

### Dynamic Plugin Loading
Using Go plugins or external processes. Rejected because:
- Complexity for users
- Compatibility issues
- Performance overhead
- Debugging difficulty

## Related

- ADR-002: Interface-First Design (planned)
- ADR-005: Registry Plugin Scope (planned)
