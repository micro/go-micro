---
layout: default
---

# Store

The store provides a pluggable interface for data storage in Go Micro.

## Features
- Key-value storage
- Multiple backend support

## Implementations
Supported stores include:
- Memory (default)
- File (`go-micro.dev/v5/store/file`)
- MySQL (`go-micro.dev/v5/store/mysql`)
- Postgres (`go-micro.dev/v5/store/postgres`)
- NATS JetStream KV (`go-micro.dev/v5/store/nats-js-kv`)

Plugins are scoped under `go-micro.dev/v5/store/<plugin>`.

Configure the store in code or via environment variables.

## Example Usage

Here's how to use the store in your Go Micro service:

```go
package main

import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/store"
    "log"
)

func main() {
    service := micro.NewService()
    service.Init()

    // Write a record
    if err := store.Write(&store.Record{Key: "foo", Value: []byte("bar")}); err != nil {
        log.Fatal(err)
    }

    // Read a record
    recs, err := store.Read("foo")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Read value: %s", string(recs[0].Value))
}
```

## Configure a specific store in code

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

## Configure via environment

```bash
MICRO_STORE=postgres MICRO_STORE_ADDRESS=postgres://user:pass@127.0.0.1:5432/db \
MICRO_STORE_DATABASE=micro MICRO_STORE_TABLE=micro \
go run main.go
```

Common variables:
- `MICRO_STORE`: selects the store implementation (`memory`, `file`, `mysql`, `postgres`, `nats-js-kv`).
- `MICRO_STORE_ADDRESS`: connection/address string for the store (plugin-specific format).
- `MICRO_STORE_DATABASE`: logical database or namespace (plugin-specific).
- `MICRO_STORE_TABLE`: logical table/bucket (plugin-specific).
