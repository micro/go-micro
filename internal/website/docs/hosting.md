---
layout: default
title: Hosting
---

# Hosting Go Micro Services

This document outlines what hosting looks like for go-micro services, the options available today, and what an ideal hosting platform would provide.

## Overview

Go Micro services are compiled Go binaries that communicate via RPC and event-driven messaging. Hosting them requires infrastructure that supports service discovery, inter-service communication, persistent storage, and configuration management. Because go-micro uses a pluggable architecture, the hosting environment can range from a single VPS to a fully orchestrated cluster.

## Current Hosting Options

### Single VPS or Bare Metal

The simplest approach. Deploy compiled binaries to a Linux server and manage them with systemd. This is the model described in the [Deployment Guide](deployment.md).

**Good for:** Small teams, early-stage projects, predictable workloads.

```
Server
├── micro@users.service
├── micro@posts.service
├── micro@web.service
└── mdns for discovery
```

- Use `micro deploy` to push binaries over SSH
- systemd handles process supervision and restarts
- mDNS provides zero-configuration service discovery on the local host
- Environment files supply per-service configuration

### Multiple Servers

Run services across several machines. This requires replacing mDNS with a network-aware registry like Consul or Etcd so services can discover each other across hosts.

```bash
# Point all services at a shared registry
MICRO_REGISTRY=consul MICRO_REGISTRY_ADDRESS=consul.internal:8500
```

- Deploy with `micro deploy` to each target server
- Use a central registry (Consul, Etcd, or NATS) for cross-host discovery
- Place a load balancer or API gateway in front of public-facing services

### Containers and Kubernetes

Package each service as a Docker image and deploy to a Kubernetes cluster or a simpler container runtime like Docker Compose.

**Dockerfile example:**

```dockerfile
FROM golang:1.21-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o service ./cmd/service

FROM alpine:3.19
COPY --from=build /app/service /service
ENTRYPOINT ["/service"]
```

**Kubernetes considerations:**

- Use the Kubernetes registry plugin or run Consul/Etcd as a StatefulSet
- ConfigMaps and Secrets replace environment files
- Kubernetes Services and Ingress handle external traffic
- Horizontal Pod Autoscaler manages scaling
- Liveness and readiness probes map to go-micro health checks

### Platform as a Service (PaaS)

Deploy to managed platforms like Railway, Render, or Fly.io. Each service runs as a separate application.

- Configuration via platform-provided environment variables
- Managed TLS and load balancing out of the box
- Use NATS or a hosted registry for service discovery between apps
- Limited control over networking and co-location

## What a Hosting Platform Needs

A purpose-built platform for go-micro services would integrate with the framework's core abstractions rather than treating services as generic containers.

### Service Discovery

The platform must run or integrate with a supported registry so services find each other automatically.

| Environment | Recommended Registry |
|---|---|
| Single host | mDNS (default, zero config) |
| Multi-host / cloud | Consul, Etcd, or NATS |
| Kubernetes | Kubernetes registry plugin |

### RPC and Messaging

Services communicate over RPC (request/response) and asynchronous messaging (pub/sub). The platform must allow direct service-to-service communication on the configured transport.

- **Transport:** HTTP (default), gRPC, or NATS
- **Broker:** HTTP event broker (default), NATS, or RabbitMQ
- Internal traffic should stay on a private network
- External traffic flows through a gateway or load balancer

### Configuration Management

Each service loads configuration from environment variables, files, or remote sources. The platform should provide:

- Per-service environment variables or config files
- Secret management with restricted access
- Hot-reload support for dynamic configuration changes

### Data Storage

go-micro's store interface supports multiple backends. The platform should provide or connect to durable storage.

- **Development:** In-memory store (default)
- **Production:** Postgres, MySQL, Redis, or other supported backends
- Persistent volumes or managed database services for stateful data

### Health Checks and Observability

The platform should monitor service health and provide visibility into behavior.

- **Health endpoints** for liveness and readiness
- **Structured logs** collected and searchable
- **Metrics** (request rates, latencies, error rates) scraped or pushed
- **Distributed tracing** across service boundaries

See [Observability](observability.md) for details on logs, metrics, and traces.

### Security

- TLS for all inter-service communication
- Service-level authentication and authorization via go-micro's auth interface
- Network isolation between services and the public internet
- Secret rotation and audit logging

### Scaling

- Horizontal scaling: run multiple instances of a service behind the client-side load balancer
- The registry tracks all instances; the selector distributes requests
- Auto-scaling based on resource usage or request volume

## Ideal Platform Architecture

A hosting platform tailored for go-micro would look like this:

```
                    ┌──────────────┐
    Internet ──────▶│   Gateway    │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
        ┌─────▼────┐ ┌────▼─────┐ ┌───▼──────┐
        │ Service A │ │ Service B│ │ Service C │
        │ (n inst.) │ │ (n inst.)│ │ (n inst.) │
        └─────┬────┘ └────┬─────┘ └───┬──────┘
              │            │            │
    ┌─────────▼────────────▼────────────▼─────────┐
    │              Private Network                 │
    │  ┌──────────┐  ┌───────┐  ┌──────────────┐  │
    │  │ Registry │  │ Broker│  │   Store      │  │
    │  │(Consul/  │  │(NATS/ │  │(Postgres/    │  │
    │  │ Etcd)    │  │ Redis)│  │ MySQL/Redis) │  │
    │  └──────────┘  └───────┘  └──────────────┘  │
    └─────────────────────────────────────────────┘
```

### Platform Capabilities

1. **Deploy** — Push binaries or container images; the platform registers them with the registry
2. **Discover** — Built-in registry so services find each other without manual configuration
3. **Route** — Gateway for external traffic; direct RPC for internal traffic
4. **Scale** — Add or remove instances; the registry and selector handle rebalancing
5. **Configure** — Environment variables, secrets, and dynamic config per service
6. **Observe** — Centralized logs, metrics dashboards, and trace visualization
7. **Secure** — Automatic TLS, service identity, and network policies

### Deployment Workflow

```
Developer                        Platform
────────                        ────────
micro build           ─────▶   Receive binary/image
micro deploy prod     ─────▶   Place on compute
                               Register with discovery
                               Start health checks
                               Route traffic
```

## Choosing a Hosting Strategy

| Factor | Single VPS | Multi-Server | Kubernetes | PaaS |
|---|---|---|---|---|
| Complexity | Low | Medium | High | Low |
| Cost | Low | Medium | High | Variable |
| Scaling | Manual | Manual | Automatic | Automatic |
| Service discovery | mDNS | Consul/Etcd/NATS | Plugin or Consul | External |
| Ops overhead | Minimal | Moderate | Significant | Minimal |
| Best for | Prototypes, small apps | Growing teams | Large-scale production | Quick launches |

## Getting Started

1. **Start simple** — Deploy to a single server with `micro deploy` and mDNS
2. **Add a registry** — When you need multiple servers, switch to Consul or Etcd
3. **Containerize** — When you need reproducible environments, add Docker
4. **Orchestrate** — When you need auto-scaling and self-healing, move to Kubernetes or a PaaS

## Related

- [Deployment](deployment.md) — Deploy services to a Linux server with systemd
- [Registry](registry.md) — Service discovery backends
- [Architecture](architecture.md) — Go Micro design and components
- [Observability](observability.md) — Logs, metrics, and tracing
- [Performance](performance.md) — Performance characteristics and tuning
