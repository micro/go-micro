---
title: "Services"
params:
  docs_main: true
  exclude_search: false
---

A service exposes Go methods over RPC. Go Micro registers it so other programs
can find it by name.

The [getting-started guide](quickstart.md) builds a notes service and calls it
through RPC, HTTP, and MCP. Continue with that service here.

## Call from Go

Other Go programs can call the same endpoint using a client:

```go
req := service.Client().NewRequest("notes", "Notes.List", &ListRequest{})
var rsp ListResponse
if err := service.Client().Call(ctx, req, &rsp); err != nil {
    return err
}
```

## Discovery

The default registry uses multicast DNS. Processes must be on the same network
with multicast enabled. A registry such as NATS works where multicast is blocked.

For NATS, run a NATS server and add this import to your service:

```go
import _ "go-micro.dev/v6/cmd/defaults"
```

Then select the same registry for each process:

```sh
export MICRO_REGISTRY=nats
export MICRO_REGISTRY_ADDRESS=127.0.0.1:4222
go run .
```

Set those variables in each terminal running `micro call`, `micro chat`, or
`micro gateway`, and in your MCP client's process environment. The CLI already
includes the registry plugins. This changes discovery; the service still uses RPC.
For a custom Go agent, pass the registry and client explicitly as shown in the
[agent guide](agents.md#discovery).

## Contracts

Go handlers work without code generation. For shared protobuf contracts, use
[protoc-gen-micro](https://github.com/micro/go-micro/tree/master/cmd/protoc-gen-micro).

**Next: [give an agent access to this service →](agents.md)**
