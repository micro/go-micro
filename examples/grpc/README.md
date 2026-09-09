# gRPC Example

This example shows how to run a go-micro v6 service as a **standard gRPC-compatible
server**: reflection is enabled, so `grpcurl` and other standard gRPC tooling can
discover and call it, and the client is a plain `google.golang.org/grpc` client
with no go-micro SDK.

This is the setup behind [micro/go-micro#4880](https://github.com/micro/go-micro/issues/4880).

## How it works

go-micro's gRPC server routes every call by parsing the standard gRPC method path
(`/helloworld.Say/Hello`) and dispatches it to the matching go-micro handler. The
`grpcserver.Reflection()` option additionally registers each handler with gRPC's
reflection service, so tools can `list` and `describe` the service without
injecting a raw `grpc.Server`.

## Running

Start the server:

```bash
cd examples/grpc
go run .
```

In another terminal, call it with the standard gRPC client:

```bash
cd examples/grpc
go run ./client --name Alice
# Response: Hello Alice
```

Or with grpcurl:

```bash
grpcurl -plaintext localhost:8080 list
# helloworld.Say
grpcurl -plaintext -d '{"name":"World"}' localhost:8080 helloworld.Say.Hello
# { "message": "Hello World" }
```

## Service name

The registry name (`helloworld`) lives on the gRPC server via
`server.Name("helloworld")`. Pass it there rather than only to
`micro.NewService("helloworld", ...)` — `micro.NewService` prepends its `Name`
option, which gets discarded when the default server is swapped out.

## Regenerating the proto

```bash
protoc -I proto \
  --go_out=proto --go_opt=paths=source_relative \
  --micro_out=proto --micro_opt=paths=source_relative \
  --go-grpc_out=proto --go-grpc_opt=paths=source_relative \
  proto/helloworld.proto
```

`protoc-gen-micro` must be v6: `go install go-micro.dev/v6/cmd/protoc-gen-micro@latest`.
