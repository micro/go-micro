---
title: "Getting Started"
description: "Build a service, use it through an agent, and extend your application in Go."
---

Go Micro connects services through discovery, RPC, and publish/subscribe. Agents
use service endpoints as tools. Workflows coordinate defined sequences of work.
You can use these parts independently and extend them through Go interfaces.

## Build your first project

Follow [Build your first project](../quickstart.md) for the complete path:
install the released CLI, build a service, and call it over RPC without model
credentials. The service-and-agent scaffold below is currently available only
from `master`.

## Keep developing locally

From a project directory, `micro run` discovers services, builds them, and starts
a local gateway and development console. It watches for source changes and
rebuilds affected services:

```sh
micro run
```

Stop any copies you launched yourself before starting the same services here.
Run with the provider configuration required by your agents. CLI chat settings
are project-specific and are not automatically the configuration of your Go
programs. See the [CLI reference](https://github.com/micro/go-micro/blob/master/cmd/micro/README.md)
for runtime flags and provider environment variables.

## Define the application in Go

The CLI is one way to use the framework. Its services, models, tools, and agent
loop are also available to your own Go programs.

- [Build a service](../client-server.md): implement typed handlers and call them over RPC.
- [Build an agent](../guides/your-first-agent.md): give it instructions and services to use as tools.
- [Compose a workflow](../guides/agents-and-workflows.md): combine service calls and agent decisions into defined steps.
- [Choose infrastructure](../architecture/index.md): understand the registry, transport, broker, store, and other replaceable components.
- [Deploy](../deployment.md): run your application beyond local development.

Use a matching framework version when building hosts for the development CLI:

```sh
go get go-micro.dev/v6@master
```

For a provider-free service-and-agent example, run
[`examples/first-agent`](https://github.com/micro/go-micro/tree/master/examples/first-agent).
