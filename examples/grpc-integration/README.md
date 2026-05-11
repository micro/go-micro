# gRPC Integration Example

Using go-micro with gRPC as the transport layer.

## What It Shows

- gRPC server with reflection-based handler registration (no protobuf compilation)
- gRPC client with retries and timeouts
- JSON codec over gRPC (protobuf optional)
- Raw bytes codec for dynamic payloads

## Run It

```bash
go run .
```

This starts a gRPC server on `:9004`, runs client demos, then keeps the server running.

## How It Works

The key insight: **handler code is identical** between default RPC and gRPC. Only the service setup changes:

```go
import (
    grpccli "go-micro.dev/v5/client/grpc"
    grpcsrv "go-micro.dev/v5/server/grpc"
)

svc := micro.New("echo",
    micro.Server(grpcsrv.NewServer()),
    micro.Client(grpccli.NewClient()),
)
```

Your handlers don't change at all:

```go
func (e *Echo) Call(ctx context.Context, req *EchoRequest, rsp *EchoResponse) error {
    rsp.Message = "echo: " + req.Message
    return nil
}
```

## Client Usage

```go
cli := grpccli.NewClient()

req := cli.NewRequest("echo", "Echo.Call", &EchoRequest{
    Message: "hello",
}, client.WithContentType("application/json"))

var rsp EchoResponse
err := cli.Call(ctx, req, &rsp, client.WithRetries(3))
```

## When to Use gRPC

| Feature | Default RPC | gRPC |
|---------|-------------|------|
| Setup | Zero config | Import grpc packages |
| Codec | JSON | JSON, Proto, Bytes |
| Streaming | Basic | Full bidirectional |
| Interop | Go Micro only | Any gRPC client |
| Performance | Good | Better for large payloads |

Use gRPC when you need interop with non-Go-Micro clients, protobuf encoding, or bidirectional streaming.
