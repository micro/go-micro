# Go Micro

Go Micro is a Go framework for distributed systems development.

It provides RPC, service discovery, messaging, storage, and configuration.
Agents use services as tools; workflows coordinate service calls and agent tasks.
The gateway exposes services through HTTP and MCP for other applications and agents.
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

- [Build and run a service](https://go-micro.dev/docs/quickstart.html) — Go code and RPC; no model needed.
- [Expose services to agents](https://go-micro.dev/docs/guides/services-with-claude-code.html) — connect your existing agent through MCP.
- [Chat with services](https://go-micro.dev/docs/guides/chat-with-services.html) — use Go Micro’s agent from the terminal.

Choose what you need. Chat does not require the gateway.

## Documentation

- [Index](https://go-micro.dev/docs/)
- [Protobuf](cmd/protoc-gen-micro/README.md)
- [PKG.go.dev](https://pkg.go.dev/go-micro.dev/v6)

## License

Apache 2.0
