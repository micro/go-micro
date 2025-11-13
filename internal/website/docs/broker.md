---
layout: default
---

# Broker

The broker provides pub/sub messaging for Go Micro services.

## Features
- Publish messages to topics
- Subscribe to topics
- Multiple broker implementations

## Implementations
Supported brokers include:
- HTTP (default)
- NATS (`go-micro.dev/v5/broker/nats`)
- RabbitMQ (`go-micro.dev/v5/broker/rabbitmq`)
- Memory (`go-micro.dev/v5/broker/memory`)

Plugins are scoped under `go-micro.dev/v5/broker/<plugin>`.

Configure the broker in code or via environment variables.

## Example Usage

Here's how to use the broker in your Go Micro service:

```go
package main

import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/broker"
    "log"
)

func main() {
    service := micro.NewService()
    service.Init()

    // Publish a message
    if err := broker.Publish("topic", &broker.Message{Body: []byte("hello world")}); err != nil {
        log.Fatal(err)
    }

    // Subscribe to a topic
    _, err := broker.Subscribe("topic", func(p broker.Event) error {
        log.Printf("Received message: %s", string(p.Message().Body))
        return nil
    })
    if err != nil {
        log.Fatal(err)
    }

    // Run the service
    if err := service.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Configure a specific broker in code

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

## Configure via environment

Using the built-in configuration flags/env vars (no code changes):

```
MICRO_BROKER=nats MICRO_BROKER_ADDRESS=nats://127.0.0.1:4222 micro server
```

Common variables:
- `MICRO_BROKER`: selects the broker implementation (`http`, `nats`, `rabbitmq`, `memory`).
- `MICRO_BROKER_ADDRESS`: comma-separated list of broker addresses.

Notes:
- NATS addresses should be prefixed with `nats://`.
- RabbitMQ addresses typically use `amqp://user:pass@host:5672`.
