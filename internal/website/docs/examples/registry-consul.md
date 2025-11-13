---
layout: default
---

# Service Discovery with Consul

Use Consul as the service registry.

## In code

```go
package main

import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/registry/consul"
)

func main() {
    reg := consul.NewConsulRegistry()
    svc := micro.NewService(micro.Registry(reg))
    svc.Init()
    svc.Run()
}
```

## Via environment

Run your service with env vars set:

```bash
MICRO_REGISTRY=consul MICRO_REGISTRY_ADDRESS=127.0.0.1:8500 go run main.go
```
