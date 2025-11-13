---
layout: default
---

# State with Postgres Store

Use the Postgres store for persistent key/value state.

## In code

```go
package main

import (
    "log"
    "go-micro.dev/v5"
    "go-micro.dev/v5/store"
    postgres "go-micro.dev/v5/store/postgres"
)

func main() {
    st := postgres.NewStore()
    svc := micro.NewService(micro.Store(st))
    svc.Init()

    _ = store.Write(&store.Record{Key: "foo", Value: []byte("bar")})
    recs, _ := store.Read("foo")
    log.Println("value:", string(recs[0].Value))

    svc.Run()
}
```

## Via environment

Run your service with env vars set:

```bash
MICRO_STORE=postgres \
MICRO_STORE_ADDRESS=postgres://user:pass@127.0.0.1:5432/postgres \
MICRO_STORE_DATABASE=micro \
MICRO_STORE_TABLE=micro \
go run main.go
```
