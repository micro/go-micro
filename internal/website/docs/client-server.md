---
layout: default
---

# Client/Server

Go Micro uses a client/server model for RPC communication between services.

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
    "go-micro.dev/v5"
    "log"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *struct{}, rsp *struct{Msg string}) error {
    rsp.Msg = "Hello, world!"
    return nil
}

func main() {
    service := micro.NewService(
        micro.Name("greeter"),
    )
    service.Init()
    micro.RegisterHandler(service.Server(), new(Greeter))
    if err := service.Run(); err != nil {
        log.Fatal(err)
    }
}
```
