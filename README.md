# Go Micro

Go Micro is a Go framework for distributed systems development.

It provides RPC, service discovery, messaging, storage, and configuration.
Agents use services as tools; workflows coordinate service calls and agent tasks.
Each component uses a Go interface with a default implementation that can be replaced.

## Features

- **Services:** RPC with streaming, message encoding and service discovery.
- **Agents:** models, conversation history, tools, and agent execution.
- **Workflows:** ordered steps, retries, and checkpoints.

## Getting started

Install the framework:

```sh
go get go-micro.dev/v6@latest
```

See the [quickstart](https://go-micro.dev/docs/quickstart.html) for local development,
[services guide](https://go-micro.dev/docs/services.html) for services,
and [agent guide](https://go-micro.dev/docs/agents.html) for agents.

## Documentation

- [Index](https://go-micro.dev/docs/)
- [Protobuf](cmd/protoc-gen-micro/README.md)
- [PKG.go.dev](https://pkg.go.dev/go-micro.dev/v6)

## License

Apache 2.0
