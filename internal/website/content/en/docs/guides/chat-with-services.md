---
title: "Chat with services"
params:
  exclude_search: false
---

Use Go Micro’s agent to call your services from a terminal conversation.
You do not need the gateway or an external MCP client.

## Before you start

Install the CLI:

```sh
go install go-micro.dev/v6/cmd/micro@master
export PATH="$(go env GOPATH)/bin:$PATH"
```

Keep your service running. If you need one, use the
[service quickstart](../quickstart.md). Check discovery before starting chat:

```sh
micro call notes Notes.List '{}'
```

The CLI and service must use the same registry. See
[service discovery](../services.md#discovery) if multicast is unavailable.

## Start chat

```sh
micro chat
```

This starts the built-in development agent, even when other agents are registered. On first use,
choose a provider, enter its key, and select a model. Requests use that provider
account. Ask **“What notes do I have?”** and check the service call in the tool
activity. The quickstart service returns “Review the release” and
“Update the documentation”.

To connect to a registered agent, use `micro chat NAME` or `/agent NAME` in chat;
the running agent owns its model configuration. You do not need to start a
second agent or enter its credentials in the client.

To host chat on your own service, use `micro new weather --agent` with a CLI
installed from `master`, or attach an agent using `service.Handle`.
See [New → Run → Chat](../agents.md#new--run--chat) for the scaffold and Go API.

Use `/agents` to list available agents and `/agent micro` to return to the built-in
agent. The prompt shows the active agent, and conversations stay separate.
These explicit routing controls currently require the CLI from `master`.
