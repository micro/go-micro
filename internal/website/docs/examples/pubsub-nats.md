---
layout: default
---

# Pub/Sub with NATS Broker

Use the NATS broker for pub/sub.

## In code

```go
package main

import (
    "log"
    "go-micro.dev/v5"
    "go-micro.dev/v5/broker"
    bnats "go-micro.dev/v5/broker/nats"
)

func main() {
    b := bnats.NewNatsBroker()
    svc := micro.NewService(micro.Broker(b))
    svc.Init()

    // subscribe
    _, _ = broker.Subscribe("events", func(e broker.Event) error {
        log.Printf("received: %s", string(e.Message().Body))
        return nil
    })

    // publish
    _ = broker.Publish("events", &broker.Message{Body: []byte("hello")})

    svc.Run()
}
```

## Via environment

Run your service with env vars set:

```bash
MICRO_BROKER=nats MICRO_BROKER_ADDRESS=nats://127.0.0.1:4222 go run main.go
```
