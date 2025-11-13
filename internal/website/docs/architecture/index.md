---
layout: default
---

# Architecture Decision Records

Documentation of architectural decisions made in Go Micro, following the ADR pattern.

## What are ADRs?

Architecture Decision Records (ADRs) capture important architectural decisions along with their context and consequences. They help understand why certain design choices were made.

## Index

### Available
- [ADR-001: Plugin Architecture](adr-001-plugin-architecture.md)
- [ADR-004: mDNS as Default Registry](adr-004-mdns-default-registry.md)
- [ADR-009: Progressive Configuration](adr-009-progressive-configuration.md)

### Planned

**Core Design**
- ADR-002: Interface-First Design
- ADR-003: Default Implementations

**Service Discovery**
- ADR-005: Registry Plugin Scope

**Communication**
- ADR-006: HTTP as Default Transport
- ADR-007: Content-Type Based Codecs

**Configuration**
- ADR-008: Environment Variable Support

## Status Values

- **Proposed**: Under consideration
- **Accepted**: Decision approved
- **Deprecated**: No longer recommended
- **Superseded**: Replaced by another ADR

## Contributing

To propose a new ADR:

1. Number it sequentially (check existing ADRs)
2. Follow the structure of existing ADRs
3. Include: Status, Context, Decision, Consequences, Alternatives
4. Submit a PR for discussion
5. Update status based on review

ADRs are immutable once accepted. To change a decision, create a new ADR that supersedes the old one.
