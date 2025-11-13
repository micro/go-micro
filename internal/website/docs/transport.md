---
layout: default
---

# Transport

The transport layer is responsible for communication between services.

## Features
- Pluggable transport implementations
- Secure and efficient communication

## Implementations
Supported transports include:
- HTTP (default)
- NATS (`go-micro.dev/v5/transport/nats`)
- gRPC (`go-micro.dev/v5/transport/grpc`)
- Memory (`go-micro.dev/v5/transport/memory`)

Plugins are scoped under `go-micro.dev/v5/transport/<plugin>`.

You can specify the transport when initializing your service or via env vars.

## Example Usage

Here's how to use a custom transport (e.g., gRPC) in your Go Micro service:

```go
package main

import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/transport/grpc"
)

func main() {
    t := grpc.NewTransport()
    service := micro.NewService(
        micro.Transport(t),
    )
    service.Init()
    service.Run()
}
```

NATS transport:
```go
import (
    "go-micro.dev/v5"
    tnats "go-micro.dev/v5/transport/nats"
)

func main() {
    t := tnats.NewTransport()
    service := micro.NewService(micro.Transport(t))
    service.Init()
    service.Run()
}
```

## Configure via environment

```bash
MICRO_TRANSPORT=nats MICRO_TRANSPORT_ADDRESS=nats://127.0.0.1:4222 go run main.go
```

Common variables:
- `MICRO_TRANSPORT`: selects the transport implementation (`http`, `nats`, `grpc`, `memory`).
- `MICRO_TRANSPORT_ADDRESS`: comma-separated list of transport addresses.
