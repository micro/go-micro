---
title: "Getting started"
params:
  docs_main: true
  exclude_search: false
---

Go Micro is a Go framework for services, agents, and workflows.
Services expose your application's operations. Agents use those operations as
tools. Workflows run steps in a defined order.

## Install

Install [Go 1.26 or newer](https://go.dev/doc/install), then create a project:

```sh
mkdir my-app
cd my-app
go mod init example.com/my-app
go get go-micro.dev/v6@master
```

These guides use the development version on `master`.

## Build

1. [Services](services.md) — write and call a service.
2. [Agents](agents.md) — give an agent access to that service.
3. [Workflows](workflows.md) — run a sequence of operations.

Start with **[Services →](services.md)**. It needs no model or API key.

## CLI

The CLI calls services, runs projects, and connects to agents:

```sh
go install go-micro.dev/v6/cmd/micro@master
export PATH="$(go env GOPATH)/bin:$PATH"
```

`micro chat` opens interactive chat. Without a running agent, it starts a local
development agent and prompts for a model provider. See the
[CLI reference](https://github.com/micro/go-micro/blob/master/cmd/micro/README.md)
for commands and configuration.
