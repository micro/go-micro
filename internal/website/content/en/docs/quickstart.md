---
title: "Getting started"
params:
  docs_main: true
  exclude_search: false
---

Build a Go service, call it, and make it available to agents through MCP.
You only need a model account if you use Go Micro's own agent.

## Install

With [Go 1.26 or newer](https://go.dev/doc/install):

```sh
go install go-micro.dev/v6/cmd/micro@master
export PATH="$(go env GOPATH)/bin:$PATH"
mkdir my-app
cd my-app
go mod init example.com/my-app
go get go-micro.dev/v6@master
```

This guide follows the development version (`@master`). Use the same version
of the framework and CLI.

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

## Connect another agent

In the second terminal, start the gateway:

```sh
micro gateway --address 127.0.0.1:8080 \
  --mcp-address 127.0.0.1:3000 \
  --mcp-tool notes.Notes.List
```

Add **`http://127.0.0.1:3000/mcp`** to your MCP client's server settings
using Streamable HTTP. The client must run on this machine for this local URL.
It will discover `notes.Notes.List`. Ask it to list your notes.
No model key is needed by the gateway; your client supplies the agent.

The same service is also available over HTTP:

```sh
curl -s http://127.0.0.1:8080/api/notes/Notes/List \
  -H 'Content-Type: application/json' -d '{}'
```

For a client that launches a local MCP process, use this configuration instead
of the gateway:

```json
{
  "mcpServers": {
    "notes": {
      "command": "micro",
      "args": ["mcp", "serve", "--tool", "notes.Notes.List"]
    }
  }
}
```

Keep the notes service running. `micro` must be on the client's PATH.

## Use Go Micro's agent

With the service still running:

```sh
micro chat
```

On first use, choose a provider, enter its key, and select a model.
Ask **“What notes do I have?”** and check the service call in the tool activity.
Model requests use your provider account. With no registered agent, chat starts
a local development agent; otherwise it connects to a running agent.

[Define your own agent in Go →](agents.md)

## Share beyond your machine

Loopback listeners above are for local use. When you bind a gateway listener
to a non-loopback address, authentication is enabled by default. Set
`MICRO_AUTH_TOKEN` to supply a bearer token, or use the generated token printed
at startup. Configure the same token in your MCP client's Authorization header.
Use HTTPS for remote access.

`--mcp-tool` selects the MCP operations to expose; repeat it for additional tools.
It does not restrict the HTTP API. Without it, MCP exposes all discovered tools,
including framework operations. See the [gateway reference](https://github.com/micro/go-micro/blob/master/cmd/micro/README.md#micro-gateway)
for authentication and scopes.

## Build further

- [Services](services.md): service calls, discovery, and protobuf contracts.
- [Agents](agents.md): models, instructions, and service access in Go.
- [Workflows](workflows.md): coordinate operations in a defined order.
