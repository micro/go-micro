---
layout: default
---

# Registry

The registry is responsible for service discovery in Go Micro. It allows services to register themselves and discover other services.

## Features
- Service registration and deregistration
- Service lookup
- Watch for changes

## Implementations
Go Micro supports multiple registry backends, including:
- MDNS (default)
- Consul
- Etcd
- NATS

You can configure the registry when initializing your service.

## Example Usage

Here's how to use a custom registry (e.g., Consul) in your Go Micro service:

```go
package main

import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/registry/consul"
)

func main() {
    reg := consul.NewRegistry()
    service := micro.NewService(
        micro.Registry(reg),
    )
    service.Init()
    service.Run()
}
```
