---
title: "Services"
params:
  docs_main: true
  exclude_search: false
---

A service exposes Go methods over RPC. Go Micro registers it so other programs
can find it by name.

## Write a service

In the project from [Getting started](quickstart.md), save this as `main.go`:

```go
package main

import (
    "context"
    "log"

    micro "go-micro.dev/v6"
)

type ListRequest struct{}
type ListResponse struct {
    Notes []string `json:"notes"`
}

type Notes struct{}

func (n *Notes) List(ctx context.Context, req *ListRequest, rsp *ListResponse) error {
    rsp.Notes = []string{"Review the release", "Update the documentation"}
    return nil
}

func main() {
    service := micro.NewService("notes")
    service.Init()
    if err := service.Handle(new(Notes)); err != nil {
        log.Fatal(err)
    }
    if err := service.Run(); err != nil {
        log.Fatal(err)
    }
}
```

Run it:

```sh
go run .
```

The service is named `notes`; its endpoint is `Notes.List`. This example returns
fixed notes. Replace the handler body with your application logic and storage.

## Call it

Leave the service running. In another terminal, use the CLI installed in
[Getting started](quickstart.md):

```sh
micro call notes Notes.List '{}'
```

The response contains the two notes. Discovery uses multicast DNS by default;
both processes must be on a network that allows it.

Other Go programs can call the same endpoint using a client:

```go
req := service.Client().NewRequest("notes", "Notes.List", &ListRequest{})
var rsp ListResponse
if err := service.Client().Call(ctx, req, &rsp); err != nil {
    return err
}
```

Go handlers work without code generation. For shared protobuf contracts, use
[protoc-gen-micro](https://github.com/micro/go-micro/tree/master/cmd/protoc-gen-micro).

**Next: [give an agent access to this service →](agents.md)**
