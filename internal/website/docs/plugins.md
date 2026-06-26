---
layout: default
---

# Plugins

Plugins are scoped under each interface directory within this repository. To use a plugin, import it directly from the corresponding interface subpackage and pass it to your service via options.

Common interfaces and locations:
- Registry: `go-micro.dev/v6/registry/*` (e.g. `consul`, `etcd`, `nats`, `mdns`)
- Broker: `go-micro.dev/v6/broker/*` (e.g. `nats`, `rabbitmq`, `http`, `memory`)
- Transport: `go-micro.dev/v6/transport/*` (e.g. `nats`, default `http`)
- Server: `go-micro.dev/v6/server/*` (e.g. `grpc` for native gRPC compatibility)
- Client: `go-micro.dev/v6/client/*` (e.g. `grpc` for native gRPC compatibility)
- Store: `go-micro.dev/v6/store/*` (e.g. `postgres`, `mysql`, `nats-js-kv`, `memory`)
- Auth, Cache, etc. follow the same pattern under their respective directories.

## Registry Examples

Consul:
```go
import (
    "go-micro.dev/v6"
    "go-micro.dev/v6/registry/consul"
)

func main() {
    reg := consul.NewConsulRegistry()
    svc := micro.NewService(
        micro.Registry(reg),
    )
    svc.Init()
    svc.Run()
}
```

Etcd:
```go
import (
    "go-micro.dev/v6"
    "go-micro.dev/v6/registry/etcd"
)

func main() {
    reg := etcd.NewRegistry()
    svc := micro.NewService("plugin-example", micro.Registry(reg))
    svc.Init()
    svc.Run()
}
```

## Broker Examples

NATS:
```go
import (
    "go-micro.dev/v6"
    bnats "go-micro.dev/v6/broker/nats"
)

func main() {
    b := bnats.NewNatsBroker()
    svc := micro.NewService("plugin-example", micro.Broker(b))
    svc.Init()
    svc.Run()
}
```

RabbitMQ:
```go
import (
    "go-micro.dev/v6"
    "go-micro.dev/v6/broker/rabbitmq"
)

func main() {
    b := rabbitmq.NewBroker()
    svc := micro.NewService("plugin-example", micro.Broker(b))
    svc.Init()
    svc.Run()
}
```

## Transport Example (NATS)
```go
import (
    "go-micro.dev/v6"
    tnats "go-micro.dev/v6/transport/nats"
)

func main() {
    t := tnats.NewTransport()
    svc := micro.NewService("plugin-example", micro.Transport(t))
    svc.Init()
    svc.Run()
}
```

## gRPC Server/Client (Native gRPC Compatibility)

For native gRPC compatibility (required for `grpcurl`, polyglot gRPC clients, etc.), use the gRPC server and client plugins. Note: This is different from the gRPC transport.

```go
import (
    "go-micro.dev/v6"
    grpcServer "go-micro.dev/v6/server/grpc"
    grpcClient "go-micro.dev/v6/client/grpc"
)

func main() {
    svc := micro.NewService(
        micro.Server(grpcServer.NewServer()),
        micro.Client(grpcClient.NewClient()),
    )
    svc.Init()
    svc.Run()
}
```

See [Native gRPC Compatibility](guides/grpc-compatibility.html) for a complete guide.

## Store Examples

Postgres:
```go
import (
    "go-micro.dev/v6"
    postgres "go-micro.dev/v6/store/postgres"
)

func main() {
    st := postgres.NewStore()
    svc := micro.NewService("plugin-example", micro.Store(st))
    svc.Init()
    svc.Run()
}
```

NATS JetStream KV:
```go
import (
    "go-micro.dev/v6"
    natsjskv "go-micro.dev/v6/store/nats-js-kv"
)

func main() {
    st := natsjskv.NewStore()
    svc := micro.NewService("plugin-example", micro.Store(st))
    svc.Init()
    svc.Run()
}
```

## Notes
- Defaults: If you don’t set an implementation, Go Micro uses sensible in-memory or local defaults (e.g., mDNS for registry, HTTP transport, memory broker/store).
- Options: Each plugin exposes constructor options to configure addresses, credentials, TLS, etc.
- Imports: Only import the plugin you need; this keeps binaries small and dependencies explicit.
