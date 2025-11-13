---
layout: default
---

# Migrating from gRPC

Step-by-step guide to migrating existing gRPC services to Go Micro.

## Why Migrate?

Go Micro adds:
- Built-in service discovery
- Client-side load balancing
- Pub/sub messaging
- Multiple transport options
- Unified tooling

You keep:
- Your proto definitions
- gRPC performance (via gRPC transport)
- Type safety
- Streaming support

## Migration Strategy

### Phase 1: Parallel Running
Run Go Micro alongside existing gRPC services

### Phase 2: Gradual Migration
Migrate services one at a time

### Phase 3: Complete Migration
All services on Go Micro

## Step-by-Step Migration

### 1. Existing gRPC Service

```protobuf
// proto/hello.proto
syntax = "proto3";

package hello;
option go_package = "./proto;hello";

service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

```go
// Original gRPC server
package main

import (
    "context"
    "log"
    "net"
    "google.golang.org/grpc"
    pb "myapp/proto"
)

type server struct {
    pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Hello " + req.Name}, nil
}

func main() {
    lis, _ := net.Listen("tcp", ":50051")
    s := grpc.NewServer()
    pb.RegisterGreeterServer(s, &server{})
    log.Fatal(s.Serve(lis))
}
```

### 2. Generate Go Micro Code

Update your proto generation:

```bash
# Install protoc-gen-micro
go install go-micro.dev/v5/cmd/protoc-gen-micro@latest

# Generate both gRPC and Go Micro code
protoc --proto_path=. \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  --micro_out=. --micro_opt=paths=source_relative \
  proto/hello.proto
```

This generates:
- `hello.pb.go` - Protocol Buffers types
- `hello_grpc.pb.go` - gRPC client/server (keep for compatibility)
- `hello.pb.micro.go` - Go Micro client/server (new)

### 3. Migrate Server to Go Micro

```go
// Go Micro server
package main

import (
    "context"
    "go-micro.dev/v5"
    "go-micro.dev/v5/server"
    pb "myapp/proto"
)

type Greeter struct{}

func (s *Greeter) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) error {
    rsp.Message = "Hello " + req.Name
    return nil
}

func main() {
    svc := micro.NewService(
        micro.Name("greeter"),
    )
    svc.Init()

    pb.RegisterGreeterHandler(svc.Server(), new(Greeter))

    if err := svc.Run(); err != nil {
        log.Fatal(err)
    }
}
```

**Key differences:**
- No manual port binding (Go Micro handles it)
- Automatic service registration
- Returns error, response via pointer parameter

### 4. Migrate Client

**Original gRPC client:**
```go
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
defer conn.Close()

client := pb.NewGreeterClient(conn)
rsp, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "John"})
```

**Go Micro client:**
```go
svc := micro.NewService(micro.Name("client"))
svc.Init()

client := pb.NewGreeterService("greeter", svc.Client())
rsp, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "John"})
```

**Benefits:**
- No hardcoded addresses
- Automatic service discovery
- Client-side load balancing
- Automatic retries

### 5. Keep gRPC Transport (Optional)

Use gRPC as the underlying transport:

```go
import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/client"
    "go-micro.dev/v5/server"
    grpcclient "go-micro.dev/v5/client/grpc"
    grpcserver "go-micro.dev/v5/server/grpc"
)

svc := micro.NewService(
    micro.Name("greeter"),
    micro.Client(grpcclient.NewClient()),
    micro.Server(grpcserver.NewServer()),
)
```

This gives you:
- gRPC performance
- Go Micro features (discovery, load balancing)
- Compatible with existing gRPC clients

## Streaming Migration

### Original gRPC Streaming

```protobuf
service Greeter {
  rpc StreamHellos (stream HelloRequest) returns (stream HelloReply) {}
}
```

```go
func (s *server) StreamHellos(stream pb.Greeter_StreamHellosServer) error {
    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }
        
        stream.Send(&pb.HelloReply{Message: "Hello " + req.Name})
    }
}
```

### Go Micro Streaming

```go
func (s *Greeter) StreamHellos(ctx context.Context, stream server.Stream) error {
    for {
        var req pb.HelloRequest
        if err := stream.Recv(&req); err != nil {
            return err
        }
        
        if err := stream.Send(&pb.HelloReply{Message: "Hello " + req.Name}); err != nil {
            return err
        }
    }
}
```

## Service Discovery Migration

### Before (gRPC with Consul)

```go
// Manually register with Consul
config := api.DefaultConfig()
config.Address = "consul:8500"
client, _ := api.NewClient(config)

reg := &api.AgentServiceRegistration{
    ID:      "greeter-1",
    Name:    "greeter",
    Address: "localhost",
    Port:    50051,
}
client.Agent().ServiceRegister(reg)

// Cleanup on shutdown
defer client.Agent().ServiceDeregister("greeter-1")
```

### After (Go Micro)

```go
import "go-micro.dev/v5/registry/consul"

reg := consul.NewConsulRegistry()
svc := micro.NewService(
    micro.Name("greeter"),
    micro.Registry(reg),
)

// Registration automatic on Run()
// Deregistration automatic on shutdown
svc.Run()
```

## Load Balancing Migration

### Before (gRPC with custom LB)

```go
// Need external load balancer or custom implementation
// Example: round-robin DNS, Envoy, nginx
```

### After (Go Micro)

```go
import "go-micro.dev/v5/selector"

// Client-side load balancing built-in
svc := micro.NewService(
    micro.Selector(selector.NewSelector(
        selector.SetStrategy(selector.RoundRobin),
    )),
)
```

## Gradual Migration Path

### 1. Start with New Services

New services use Go Micro, existing services stay on gRPC.

```go
// New Go Micro service can call gRPC services
// Configure gRPC endpoints directly
grpcConn, _ := grpc.Dial("old-service:50051", grpc.WithInsecure())
oldClient := pb.NewOldServiceClient(grpcConn)
```

### 2. Migrate Read-Heavy Services First

Services with many clients benefit most from service discovery.

### 3. Migrate Services with Fewest Dependencies

Leaf services are easier to migrate.

### 4. Add Adapters if Needed

```go
// gRPC adapter for Go Micro service
type GRPCAdapter struct {
    microClient pb.GreeterService
}

func (a *GRPCAdapter) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    return a.microClient.SayHello(ctx, req)
}

// Register adapter as gRPC server
s := grpc.NewServer()
pb.RegisterGreeterServer(s, &GRPCAdapter{microClient: microClient})
```

## Checklist

- [ ] Update proto generation to include `--micro_out`
- [ ] Convert handler signatures (response via pointer)
- [ ] Replace `grpc.Dial` with Go Micro client
- [ ] Configure service discovery (Consul, Etcd, etc)
- [ ] Update deployment (remove hardcoded ports)
- [ ] Update monitoring (Go Micro metrics)
- [ ] Test service-to-service communication
- [ ] Update documentation
- [ ] Train team on Go Micro patterns

## Common Issues

### Port Already in Use

**gRPC**: Manual port management
```go
lis, _ := net.Listen("tcp", ":50051")
```

**Go Micro**: Automatic or explicit
```go
// Let Go Micro choose
svc := micro.NewService(micro.Name("greeter"))

// Or specify
svc := micro.NewService(
    micro.Name("greeter"),
    micro.Address(":50051"),
)
```

### Service Not Found

Check registry:
```bash
# Consul
curl http://localhost:8500/v1/catalog/services

# Or use micro CLI
micro services
```

### Different Serialization

gRPC uses protobuf by default. Go Micro supports multiple codecs.

Ensure both use protobuf:
```go
import "go-micro.dev/v5/codec/proto"

svc := micro.NewService(
    micro.Codec("application/protobuf", proto.Marshaler{}),
)
```

## Performance Comparison

| Scenario | gRPC | Go Micro (HTTP) | Go Micro (gRPC) |
|----------|------|----------------|-----------------|
| Simple RPC | ~25k req/s | ~20k req/s | ~24k req/s |
| With Discovery | N/A | ~18k req/s | ~22k req/s |
| Streaming | ~30k msg/s | ~15k msg/s | ~28k msg/s |

*Go Micro with gRPC transport performs similarly to pure gRPC*

## Next Steps

- Read [Go Micro Architecture](../architecture.md)
- Explore [Plugin System](../plugins.md)
- Check [Production Patterns](../examples/realworld/)

## Need Help?

- [Examples](../examples/)
- [GitHub Issues](https://github.com/micro/go-micro/issues)
- [API Documentation](https://pkg.go.dev/go-micro.dev/v5)
