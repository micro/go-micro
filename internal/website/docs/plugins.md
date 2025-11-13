---
layout: default
---

# Plugins

Plugins are scoped under each interface directory within this repository. To use a plugin, import it directly from the corresponding interface subpackage and pass it to your service via options.

Common interfaces and locations:
- Registry: `go-micro.dev/v5/registry/*` (e.g. `consul`, `etcd`, `nats`, `mdns`)
- Broker: `go-micro.dev/v5/broker/*` (e.g. `nats`, `rabbitmq`, `http`, `memory`)
- Transport: `go-micro.dev/v5/transport/*` (e.g. `nats`, default `http`)
- Store: `go-micro.dev/v5/store/*` (e.g. `postgres`, `mysql`, `nats-js-kv`, `memory`)
- Auth, Cache, etc. follow the same pattern under their respective directories.

## Registry Examples

Consul:
```go
import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/registry/consul"
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
    "go-micro.dev/v5"
    "go-micro.dev/v5/registry/etcd"
)

func main() {
    reg := etcd.NewRegistry()
    svc := micro.NewService(micro.Registry(reg))
    svc.Init()
    svc.Run()
}
```

## Broker Examples

NATS:
```go
import (
    "go-micro.dev/v5"
    bnats "go-micro.dev/v5/broker/nats"
)

func main() {
    b := bnats.NewNatsBroker()
    svc := micro.NewService(micro.Broker(b))
    svc.Init()
    svc.Run()
}
```

RabbitMQ:
```go
import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/broker/rabbitmq"
)

func main() {
    b := rabbitmq.NewBroker()
    svc := micro.NewService(micro.Broker(b))
    svc.Init()
    svc.Run()
}
```

## Transport Example (NATS)
```go
import (
    "go-micro.dev/v5"
    tnats "go-micro.dev/v5/transport/nats"
)

func main() {
    t := tnats.NewTransport()
    svc := micro.NewService(micro.Transport(t))
    svc.Init()
    svc.Run()
}
```

## Store Examples

Postgres:
```go
import (
    "go-micro.dev/v5"
    postgres "go-micro.dev/v5/store/postgres"
)

func main() {
    st := postgres.NewStore()
    svc := micro.NewService(micro.Store(st))
    svc.Init()
    svc.Run()
}
```

NATS JetStream KV:
```go
import (
    "go-micro.dev/v5"
    natsjskv "go-micro.dev/v5/store/nats-js-kv"
)

func main() {
    st := natsjskv.NewStore()
    svc := micro.NewService(micro.Store(st))
    svc.Init()
    svc.Run()
}
```

## Notes
- Defaults: If you don’t set an implementation, Go Micro uses sensible in-memory or local defaults (e.g., mDNS for registry, HTTP transport, memory broker/store).
- Options: Each plugin exposes constructor options to configure addresses, credentials, TLS, etc.
- Imports: Only import the plugin you need; this keeps binaries small and dependencies explicit.
