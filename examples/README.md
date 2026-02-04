# Go Micro Examples

This directory contains runnable examples demonstrating various go-micro features and patterns.

## Quick Start

Each example can be run with `go run .` from its directory.

## Examples

### [hello-world](./hello-world/)
Basic RPC service demonstrating core concepts:
- Service creation and registration
- Handler implementation
- Client calls
- Health checks

**Run it:**
```bash
cd hello-world
go run .
```

### [web-service](./web-service/)
HTTP web service with service discovery:
- HTTP handlers
- Service registration
- Health checks
- JSON REST API

**Run it:**
```bash
cd web-service
go run .
```

## Coming Soon

The following examples are planned:

- **pubsub-events** - Event-driven architecture with NATS
- **grpc-integration** - Using go-micro with gRPC
- **production-ready** - Complete production-grade service with observability

## Prerequisites

Some examples require external dependencies:

- **NATS**: `docker run -p 4222:4222 nats:latest`
- **Consul**: `docker run -p 8500:8500 consul:latest agent -dev -ui -client=0.0.0.0`
- **Redis**: `docker run -p 6379:6379 redis:latest`

## Contributing

To add a new example:

1. Create a new directory
2. Add a descriptive README.md
3. Include working code with comments
4. Add to this index
5. Ensure it runs with `go run .`

