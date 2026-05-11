# Pubsub Events Example

Event-driven architecture with the **broker** and **events** packages.

## Two Patterns

| Pattern | Package | Persistence | Use Case |
|---------|---------|-------------|----------|
| **Broker** | `broker` | No (fire-and-forget) | Notifications, cache invalidation |
| **Events** | `events` | Yes (durable stream) | Order processing, audit logs |

## Run It

```bash
go run .
```

This runs three demos in sequence:

1. **Broker demo** — publish/subscribe with queue groups
2. **Events demo** — durable streaming with metadata
3. **Service** — a notification service that publishes events when called

## How It Works

### Broker (fire-and-forget)

```go
// Publish
broker.Publish("user.created", &broker.Message{
    Body: jsonBytes,
})

// Subscribe
broker.Subscribe("user.created", func(e broker.Event) error {
    // handle message
    return nil
})

// Queue group (messages split across consumers)
broker.Subscribe("user.created", handler, broker.Queue("workers"))
```

### Events (durable streaming)

```go
stream, _ := events.NewStream()

// Publish with metadata
stream.Publish("order.placed", order, events.WithMetadata(map[string]string{
    "user_id": order.UserID,
}))

// Consume with consumer group
ch, _ := stream.Consume("order.placed", events.WithGroup("processors"))
for ev := range ch {
    var order OrderPlaced
    ev.Unmarshal(&order)
    // process...
}
```

## Production Setup

For production, swap the in-memory implementations for NATS:

```go
import (
    natsbroker "go-micro.dev/v5/broker/nats"
    "go-micro.dev/v5/events/natsjs"
)

// Broker with NATS
micro.New("myservice", micro.Broker(natsbroker.NewBroker(
    natsbroker.Addrs("nats://localhost:4222"),
)))

// Events with NATS JetStream (durable, persistent)
stream, _ := natsjs.NewStream(natsjs.Address("localhost:4222"))
```
