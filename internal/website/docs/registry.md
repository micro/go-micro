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

## Plugins Location

Registry plugins live in this repository under `go-micro.dev/v5/registry/<plugin>` (e.g., `consul`, `etcd`, `nats`). Import the desired package and pass it via `micro.Registry(...)`.

## Configure via environment

```
MICRO_REGISTRY=etcd MICRO_REGISTRY_ADDRESS=127.0.0.1:2379 micro server
```

Common variables:
- `MICRO_REGISTRY`: selects the registry implementation (`mdns`, `consul`, `etcd`, `nats`).
- `MICRO_REGISTRY_ADDRESS`: comma-separated list of registry addresses.

Backend-specific variables:
- Etcd: `ETCD_USERNAME`, `ETCD_PASSWORD` for authenticated clusters.

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
