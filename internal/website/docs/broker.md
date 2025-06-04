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
- Memory (default)
- NATS
- RabbitMQ

Configure the broker when creating your service as needed.

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
