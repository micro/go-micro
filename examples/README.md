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

### [pubsub-events](./pubsub-events/)
Event-driven architecture with NATS:
- Publishing events
- Subscribing to topics
- Event handlers
- Asynchronous processing

**Run it:**
```bash
cd pubsub-events
go run publisher/main.go  # Terminal 1
go run subscriber/main.go # Terminal 2
```

### [web-service](./web-service/)
HTTP web service with service discovery:
- HTTP handlers
- Service registration
- Health checks
- Static file serving

**Run it:**
```bash
cd web-service
go run .
```

### [grpc-integration](./grpc-integration/)
Using go-micro with gRPC:
- Protocol buffer definitions
- gRPC client/server
- Code generation
- Type-safe APIs

**Run it:**
```bash
cd grpc-integration
make proto  # Generate code
go run server/main.go  # Terminal 1
go run client/main.go  # Terminal 2
```

### [production-ready](./production-ready/)
Complete production-grade service:
- Structured logging
- Metrics and tracing
- Health checks
- Graceful shutdown
- Configuration management
- Error handling

**Run it:**
```bash
cd production-ready
go run .
```

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

