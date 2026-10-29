---
title: "Agents"
params:
  docs_main: true
  exclude_search: false
---

An agent combines a model, instructions, and tools. Service endpoints become its
tools, so you can use the same operations from code and conversation.

## New → Run → Chat

Install the CLI from `master` until this feature is released:

```sh
go install go-micro.dev/v6/cmd/micro@master
```

The released v6.15.1 CLI supports the service-only [getting-started
path](quickstart.md), but does not include this combined scaffold. Check the
binary you are about to use with `micro --version`.

Create and run a service with an agent:

```sh
micro new weather --agent
cd weather
export OPENAI_API_KEY=your-key # or another supported provider key
micro run
```

In another terminal:

```sh
micro call weather Weather.Call '{"name":"Alice"}'
micro chat weather
```

Ask it to greet Alice. The generated `Weather.Call` handler returns “Hello Alice”.
The name does not generate weather functionality: add your own methods in
`handler/` to implement that. The agent uses those methods as tools.

The service hosts RPC and chat on the same server, with one discovery entry and
one lifecycle. Configure the provider on the service; the chat client needs no
model credentials. `micro run` builds and watches the service; `go run .` also works.
Use `micro gateway` separately when you need HTTP or MCP.

## Add an agent to a service

Register your business handlers, then attach an agent before running the service:

```go
if err := service.Handle(handler.New()); err != nil {
    log.Fatal(err)
}
if err := service.Handle(micro.NewAgent("weather")); err != nil {
    log.Fatal(err)
}
if err := service.Run(); err != nil {
    log.Fatal(err)
}
```

The service owns the agent's name, client, registry, broker, storage, and shutdown.
By default, the attached agent can use only this service's business endpoints.
Use `micro.AgentServices("weather", "notes")` to select other services as well.
Agent chat, session, task, and schedule endpoints are excluded from model tools.

`micro new --agent` uses the default Go handler template. It does not need a model
key to scaffold the project. `--prompt` is a separate, optional code-generation
path; it is not part of this workflow.

## Standalone agent

An agent can also run separately and use existing services. Keep the [notes service](services.md) running. In your project, create
`cmd/assistant/main.go`:

```go
package main

import (
    "log"
    "os"

    micro "go-micro.dev/v6"
)

func main() {
    agent := micro.NewAgent("assistant",
        micro.AgentProvider("openai"),
        micro.AgentAPIKey(os.Getenv("OPENAI_API_KEY")),
        micro.AgentServices("notes"),
        micro.AgentPrompt("Use the notes service to answer questions about notes."),
    )
    if err := agent.Run(); err != nil {
        log.Fatal(err)
    }
}
```

Start it in a second terminal with your provider key:

```sh
export OPENAI_API_KEY=your-key
go run ./cmd/assistant
```

## Use it

Wait for the agent to register, then in a third terminal:

```sh
micro chat assistant
```

This connects explicitly to `assistant`. Ask:

```text
What notes do I have?
```

The agent can call `Notes.List` and answer using its result. Check the tool
activity to see that the service was called. This uses your model provider and
incurs its normal API charges.

Use `/agents` to list available agents and `/agent NAME` to switch.
`/agent micro` returns to the built-in development agent. Each destination keeps
its own conversation. Plain `micro chat` always starts the built-in agent.

Your Go code controls the provider, instructions, and allowed services. Change
those options to build your own agent. For programmatic calls, the agent exposes
`Ask(ctx, message)`; see the [agent API](https://pkg.go.dev/go-micro.dev/v6/agent).

Configure the provider and its key in the agent process. The service and tools
stay the same when you change models; chat connects to the running agent.

## Discovery

Agents use mDNS by default. For another registry, give the agent and its client
the same registry. For NATS, import `agent`, `client`, `registry`, and
`registry/nats` from `go-micro.dev/v6`, then add these options:

```go
reg := nats.NewNatsRegistry(registry.Addrs("127.0.0.1:4222"))
a := micro.NewAgent("assistant",
    agent.WithRegistry(reg),
    agent.WithClient(client.NewClient(client.Registry(reg))),
    micro.AgentServices("notes"),
    // Your model and instructions, as above.
)
```

The CLI's `MICRO_REGISTRY` settings configure CLI processes. Library code selects
its components through Go options.

## Execution

`Ask` runs work in the caller's context. `StartTask` runs work in the agent host
and returns a task ID; `GetTask` reads its result and `StopTask` cancels it.
Scheduled work uses that same task path. Keep the host running with `Run` and
shut it down with `Stop`.

For recovery, configure `agent.WithCheckpoint(agent.StoreCheckpoint(st, "assistant"))`,
where `st` is your persistent store. `agent.Pending` lists unfinished runs and
`agent.Resume` resumes a chosen run. Recovery is explicit: a tool interrupted by
a crash may already have performed its action.

These are agent capabilities; no workflow definition is required.
[Workflows](workflows.md) remain available for applications that need a predefined
sequence of operations.

## Call an agent from Go

Use the same registry configuration as the running agent. The host owns its
model credentials and tools; the client only needs RPC access.

```go
remote := agent.NewClient("assistant", client.DefaultClient)
ctx := agent.WithSession(context.Background(), "my-conversation")
response, err := remote.Ask(ctx, "What notes do I have?")
if err != nil {
    return err
}
fmt.Println(response.Reply)
```

The packages are `go-micro.dev/v6/agent` and `go-micro.dev/v6/client`.
Reuse the session ID to continue a conversation. Use a different ID for each
conversation; the host must authenticate and authorize access separately.

For live text, tool activity and approvals, use `remote.StreamAsk(ctx, message)`.
It returns the same `agent.StreamEvent` values used by local agents:

```go
stream, err := remote.StreamAsk(ctx, "Read my notes")
if err != nil {
    return err
}
defer stream.Close()
for {
    event, err := stream.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        return err
    }
    switch event.Type {
    case agent.StreamEventToken:
        fmt.Print(event.Token)
    case agent.StreamEventApproval:
        // Ask the user here. This example refuses the action.
        if err := stream.Approve(event.ApprovalID, false); err != nil {
            return err
        }
    }
}
```

Cancelling the context or closing the stream cancels unfinished work. Use
`stream.Context()` when waiting for a user decision so a disconnect also ends
the wait. A disconnect before completion returns an error; calls are never
retried automatically because actions may already have taken effect. Streaming
requires a host with the `AgentInteraction` endpoint.
