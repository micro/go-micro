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
## Discovery failures and endpoint budgets

Missing services now return a structured 404. Unavailable discovery backends
and empty usable-node sets return 503. In-process callers can inspect
`*client.DiscoveryError`, use `errors.Is(err, registry.ErrNotFound)` or
`errors.Is(err, client.ErrNoNodes)`, and inspect the wrapped backend cause. NATS
connection failures also match `registry.ErrUnavailable`. Across RPC boundaries,
use the structured status code; backend Go error identities are local.

Configure a slow endpoint once when constructing a client:

```go
c := client.NewClient(client.EndpointOptions(map[string][]client.CallOption{
    "assistant/Agent.Chat": {
        client.WithRequestTimeout(2*time.Minute),
        client.WithConnectionTimeout(2*time.Minute),
    },
}))
```

Keys are `service/endpoint`. Endpoint options override client defaults, and
explicit per-call options override endpoint settings. This works for the default
RPC and native gRPC clients, including generated clients that wrap them. Caller
deadlines still apply: the earlier deadline wins.

The current defaults are a 30-second total request budget and a 5-second
connection/request-attempt budget. Streaming has a separate `WithStreamTimeout`.
Applications using `service.Init()` can configure the existing command/env
settings `MICRO_CLIENT_REQUEST_TIMEOUT` and `MICRO_CLIENT_CONNECTION_TIMEOUT`
with Go duration strings such as `2m`. Library-only clients can use
`client.RequestTimeout`, `client.ConnectionTimeout`, or endpoint options directly.
