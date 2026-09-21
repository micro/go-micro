# NATS JetStream

This plugin uses NATS with JetStream to send and receive events.

## Create a stream

```go
ev, err := natsjs.NewStream(
  natsjs.Address("nats://10.0.1.46:4222"),
  natsjs.MaxAge(24*160*time.Minute),
)
```

## Consume a stream

Durable streams require an explicit consumer group. A newly created durable
consumer starts at the beginning of the stream and resumes from its persisted
position on later connections. Use `events.WithOffset` to choose a different
starting time when the durable consumer is first created. When durable streams
are disabled, consumers are ephemeral and receive only newly published events
by default.

```go
ee, err := events.Consume("test",
  events.WithAutoAck(false, time.Second*30),
  events.WithGroup("testgroup"),
)
if err != nil {
  panic(err)
}
go func() {
  for {
    msg := <-ee
    // Process the message
    logger.Info("Received message:", string(msg.Payload))
    err := msg.Ack()
    if err != nil {
      logger.Error("Error acknowledging message:", err)
    } else {
      logger.Info("Message acknowledged")
    }
  }
}()

```

## Publish an Event to the stream

```go
err = ev.Publish("test", []byte("hello world"))
if err != nil {
  panic(err)
}
```


## Acknowledgements

Automatic acknowledgement on delivery is the default. Use `events.WithAutoAck(false, ackWait)`
and call `event.Ack()` after processing succeeds, or `event.Nack()` to request
redelivery after a failure. Unacknowledged events are redelivered after `ackWait`.

With `events.WithAutoAck(true, ackWait)`, events are acknowledged when received
from the channel, **before application processing completes**. A subsequent
processing failure can lose the event; use manual acknowledgements when
processing must succeed before delivery is confirmed.

## Subjects, stream configuration, and stable IDs

Topics are NATS subjects, so dotted names such as `orders.created.v1` work.
Simple legacy names keep their existing stream names. Other subjects receive a
stable name from `natsjs.StreamName(topic)`. Publishing or consuming creates a
missing stream; existing stream configuration is never silently changed.

Configure several subjects on one stream with a per-topic configuration hook:

```go
ev, err := natsjs.NewStream(
    natsjs.Address("nats://localhost:4222"),
    natsjs.SynchronousPublish(true),
    natsjs.WithStreamConfig(func(topic string) (nats.StreamConfig, error) {
        return nats.StreamConfig{
            Name: "orders", Subjects: []string{"orders.*.v1"},
            Storage: nats.FileStorage, Replicas: 1,
            MaxAge: 24*time.Hour, MaxMsgs: 100000,
            Duplicates: 2*time.Minute,
        }, nil
    }),
)
```

Use the same mapping for publishers and consumers. Each durable consumer name
is scoped to the stream, so use distinct groups for different subject filters.
The callback uses `github.com/nats-io/nats.go`'s configuration type. It controls
creation only; manage updates to existing streams through NATS explicitly.

```go
err = ev.Publish("orders.created.v1", payload, events.WithID(outboxID))
```

`WithID` preserves the event ID and sets JetStream's `Nats-Msg-Id` header. Retries
within the stream's duplicate window are deduplicated by the server. IDs must
identify an event uniquely across the entire stream, including shared subjects.
An empty ID still generates a UUID. Deduplication is time bounded; consumers
should retain their own idempotency handling for longer-lived retries. The
memory/store event stream preserves supplied IDs but does not promise the same
JetStream publish-deduplication semantics.
