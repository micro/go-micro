# GRPC Broker

The grpc broker is a point to point grpc broker.

## Overview

This broker is like the built in go-micro http2 broker except it uses grpc and protobuf as the protocol and messaging format. 
It uses the go-micro registry to subscribe to a topic, creating a service for that topic. Publishers lookup the registry 
for subscribers and publish to them, hence point to point.

## Usage

```go
import (
	"github.com/asim/go-micro/plugins/broker/grpc"
)

// create and connect (starts a grpc server)
b := grpc.NewBroker()
b.Init()
b.Connect()

// subscribe
sub, _ := b.Subscribe("events")
defer sub.Unsubscribe()

// publish
b.Publish("events", &broker.Message{
	Headers: map[string]string{"type": "event"},
	Body: []byte(`an event`),
})
```
