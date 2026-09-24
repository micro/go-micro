---
title: "Quick Start"
description: "Start a conversation, develop services, and use them as agent tools."
---


Install [Go 1.25 or newer](https://go.dev/doc/install), then the `micro` CLI:

```bash
go install go-micro.dev/v6/cmd/micro@latest
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`. Set a provider API key and
start a conversation in a new directory:

```bash
export OPENAI_API_KEY=your-api-key
mkdir my-app && cd my-app
micro chat --provider openai
```

Describe a capability you want to build, then ask the agent to use it. For example:

```text
Build a notes service that can save, list, and search notes.
Save a note: the launch review is on Friday.
Find my notes about the launch.
```

With no registered agents, `micro chat` uses its built-in development agent.
When a capability is missing, it can generate a service in the current directory,
compile and start it, and discover its endpoints as tools. You can then use that
service in the same conversation. The generated Go source is yours to inspect
and change.

Services started by this chat session stop when you exit. To continue developing
the generated project, run `micro run` from its directory: it starts the services
with hot reload, an API gateway, and a console. Edit the Go code to extend existing
services. If agents are already registered, `micro chat` routes requests to them;
`micro chat <name>` selects a particular agent.

### Start from a prompt

You can also describe the application up front:

```bash
micro run --prompt "a notes service with saving, listing, and search" --provider openai
```

Micro proposes the services for you to review, generates their Go code and an
agent, builds them, and opens a conversation with the running system.

For other providers, see [AI Integration](ai-integration/index.md). For a walkthrough without
an API key, see the [first-agent example](https://github.com/micro/go-micro/tree/master/examples/first-agent).

## Next steps

- [Getting Started](getting-started/index.md): create your own Go agent and give it services as tools.
- [Your First Agent](guides/your-first-agent.md): a complete service-backed agent walkthrough.
- [Install troubleshooting](guides/install-troubleshooting.md): check the CLI, Go toolchain, and PATH.
- [Debugging your agent](guides/debugging-agents.md): inspect tools and recorded runs.
- [Examples](examples/): runnable services, agents, and workflows.
- [Deployment](deployment.md): build and run your application in production.

For a provider-free check, `micro agent demo` prints the mock-model walkthrough;
`micro examples` and `micro zero-to-hero` print the available examples and harness
commands. See the [no-secret transcript](guides/no-secret-first-agent.md) and
[0→hero reference](guides/zero-to-hero.md). These are optional checks, not steps
required before conversational development.
