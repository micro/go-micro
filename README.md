# Go Micro [![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/go-micro.dev/v6?tab=doc)

A framework for building services and agents in Go.

## Overview

Go Micro started with service communication: RPC, discovery, and pub/sub built
on pluggable interfaces. That foundation now gives agents tools they can discover
and call. You can build services independently, add an agent to work with them,
and use workflows for repeatable sequences of work.

## Features

- **Services** implement capabilities and own their data.
- **Models** provide access to AI providers through a common interface.
- **Agents** interpret requests and use tools to carry out work.
- **Workflows** coordinate defined steps, including service calls and agent decisions.

Registry, client/server, broker, transport, codec, and store remain independently
usable. See the [architecture](internal/website/content/en/docs/architecture/index.md)
for how the pieces fit together.

## Create an agent

With [Go 1.25 or newer](https://go.dev/doc/install):

```bash
mkdir assistant && cd assistant
go mod init example.com/assistant
go get go-micro.dev/v6
export OPENAI_API_KEY=your-api-key
```

Save as `main.go`, then run `go run .`:

```go
package main

import (
    "log"
    "os"

    "go-micro.dev/v6"
)

func main() {
    assistant := micro.NewAgent("assistant",
        micro.AgentPrompt("Use your tools to carry out requests."),
        micro.AgentProvider("openai"),
        micro.AgentAPIKey(os.Getenv("OPENAI_API_KEY")),
        micro.AgentServices(), // Start without application service tools.
    )
    if err := assistant.Run(); err != nil {
        log.Fatal(err)
    }
}
```

Give it capabilities by writing services and selecting their registered names
with `micro.AgentServices("notes", "search")`. These are services you supply;
their endpoints become the agent's tools. An agent can return an answer, perform
an action, or call a service that builds and saves something for later use.

Follow [Your First Agent](internal/website/content/en/docs/guides/your-first-agent.md)
for a complete service-backed agent. Agents can be called through Go, RPC, or
the CLI.

## CLI

Install the optional CLI and talk to the agent from another terminal:

```bash
go install go-micro.dev/v6/cmd/micro@latest
```

Run the chat

```
micro chat assistant --provider openai
```

To develop through conversation, run `micro chat --provider openai` in a new
directory with your provider key exported. With no registered agents, the CLI's
development agent can generate, build, and start services, then use them as tools.
Those processes stop when chat exits; the source remains. Use `micro run` to
continue developing the project. When agents are registered, chat routes to them.

See the [Quick Start](internal/website/content/en/docs/quickstart.md) and
[CLI reference](cmd/micro/README.md) for generation, hot reload, and deployment.

## Documentation

- [Getting started](internal/website/content/en/docs/getting-started/index.md)
- [Models and providers](model/README.md)
- [Agents and workflows](internal/website/content/en/docs/guides/agents-and-workflows.md)
- [Durability and recovery](internal/website/content/en/docs/guides/durability.md)
- [Debugging agents](internal/website/content/en/docs/guides/debugging-agents.md)
- [Full documentation](https://go-micro.dev/docs/) · [Go API reference](https://pkg.go.dev/go-micro.dev/v6)

## Examples

Browse the [examples](examples/README.md), run the provider-free
[first-agent example](examples/first-agent/), or follow the
[0→hero reference](internal/website/content/en/docs/guides/zero-to-hero.md).

## Community and support

[Discord](https://discord.gg/G8Gk5j3uXr) · [Commercial support](SUPPORT.md)
