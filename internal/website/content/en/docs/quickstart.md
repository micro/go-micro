---
title: "Getting started"
params:
  docs_main: true
  exclude_search: false
---

Build and run a Go service, then call it over RPC. No model or agent is needed.

## Install

With [Go 1.26 or newer](https://go.dev/doc/install):

```sh
go install go-micro.dev/v6/cmd/micro@v6.15.1
export PATH="$(go env GOPATH)/bin:$PATH"
mkdir my-app
cd my-app
go mod init example.com/my-app
go get go-micro.dev/v6@v6.15.1
```

This guide uses v6.15.1 of both the framework and CLI.

## Write a service

Save this as `main.go`. Replace the handler body with your application's logic:

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

Start it and leave it running:

```sh
go run .
```

In a second terminal:

```sh
micro call notes Notes.List '{}'
```

The response contains `Review the release` and `Update the documentation`.
The service is named `notes`; the operation is `Notes.List`.

Discovery uses multicast DNS by default. If your network blocks multicast,
use a shared registry such as NATS; see [Services](services.md#discovery).

Your service is running and can be called by other applications. You can stop here.

## Choose what to do next

- [Expose it to Claude Code through MCP](guides/services-with-claude-code.md).
- [Talk to it with micro chat](guides/chat-with-services.md). No gateway required.

## Build further

- [Services](services.md): service calls, discovery, and protobuf contracts.
- [Agents](agents.md): define your own agent in Go.
- [Workflows](workflows.md): coordinate operations in a defined order.
