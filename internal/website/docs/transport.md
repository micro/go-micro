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
- TCP (default)
- gRPC

You can specify the transport when initializing your service.

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
