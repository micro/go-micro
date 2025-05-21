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

