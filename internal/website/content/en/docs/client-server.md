---
title: "Client/Server"
description: Go Micro uses a client/server model for RPC communication between services.
---
## Client
The client is used to make requests to other services.

## Server
The server handles incoming requests.

Both client and server are pluggable and support middleware wrappers for additional functionality.

## Example Usage

Here's how to define a simple handler and register it with a Go Micro server:

```go
package main

import (
    "context"
    "go-micro.dev/v6"
    "log"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *struct{}, rsp *struct{Msg string}) error {
    rsp.Msg = "Hello, world!"
    return nil
}

func main() {
    service := micro.NewService("greeter",
    )
    service.Init()
    micro.RegisterHandler(service.Server(), new(Greeter))
    if err := service.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Wait for a dependency

Startup order does not guarantee that a dependency has registered. Use the Go
helper with a caller deadline instead of maintaining a retry loop in each service:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := micro.WaitForService(ctx, service, "identity",
    micro.WaitBackoff(100*time.Millisecond, 2*time.Second),
); err != nil {
    return err
}
```

The helper waits for a registered node with an address. Discovery does not prove
that the application is ready to answer requests. Add `micro.WaitProbe(func(ctx
context.Context) error { ... })` to perform a read-only health or initialization
RPC after discovery. The probe may run repeatedly and should honor its context.

Registry errors, empty node lists, and probe failures are retried with capped
exponential backoff. Defaults are 100 milliseconds initially and 2 seconds at the
cap. Cancellation returns promptly even if a registry backend ignores context;
its one in-flight attempt may finish later. `errors.Is` can identify the context
cancellation/deadline and the last completed failure. This helper does not alter
service readiness endpoints or start a background dependency monitor.
