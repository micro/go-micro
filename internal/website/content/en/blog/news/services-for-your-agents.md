---
title: "Your services, available to agents"
date: 2026-10-18T11:00:00+01:00
description: "Build a service in Go and use it through RPC, HTTP, MCP, or your own agent."
category:
- news
tags:
- services
- agents
- mcp
---

A service gives a capability a name, an interface, and somewhere to run.
That remains the foundation of Go Micro.

An agent needs those capabilities too. It might need to look something up,
update a record, or start a process. The service implements the operation;
the agent decides when to use it.

Go Micro v6.15.1 takes that all the way through. Write a Go
service, call it from the command line, and expose it through the gateway.
An MCP client can discover and use the same operation. You can also use
`micro chat`, or define an agent in Go with its own model, instructions,
and service access.

You choose which MCP tools to expose. The gateway provides a connection URL,
and its HTTP and MCP listeners accept the same gateway credentials. Services
remain usable independently of an agent, and workflows can coordinate their
operations when the sequence needs to be defined in code.

Start with the [quickstart](/docs/quickstart.html), which installs v6.15.1 and
provides a complete service. Once it is running:

```sh
micro call notes Notes.List '{}'
```

The response contains two notes: “Review the release” and “Update the documentation”.
Expose that same operation to an MCP client:

```sh
micro gateway --address 127.0.0.1:8080 \
  --mcp-address 127.0.0.1:3000 \
  --mcp-tool notes.Notes.List
```

Connect your client to `http://127.0.0.1:3000/mcp`. It can discover
`notes.Notes.List` and call it to retrieve those notes. The gateway needs no
model key; the client supplies the agent. These addresses are local to your
machine. The guide covers authentication when exposing a listener beyond it.

We checked the tagged release using real service, gateway, and client processes.
That found problems unit tests had missed: registry settings that did not reach
the gateway, and MCP calls that returned empty content. Those paths now have
an automated check that builds the examples directly from the guides. The model
in that check is scripted; discovery, RPC, and the external client calls are real.

[Build a service and connect your agent →](/docs/quickstart.html)
