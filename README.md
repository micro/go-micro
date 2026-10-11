# Go Micro

A Go framework for building services, agents, and workflows.

Write your application's capabilities as services. Go Micro handles discovery,
RPC, and publish/subscribe so they can work together. Agents can discover and call
those services as tools; workflows coordinate repeatable work across them.
Use the service framework on its own, or add agents as your application grows.

## Start building

With [Go 1.26+](https://go.dev/doc/install), install the CLI and open a project:

```sh
go install go-micro.dev/v6/cmd/micro@master
export PATH="$(go env GOPATH)/bin:$PATH"
mkdir my-app && cd my-app
micro chat
```

This chat experience is on `master`, ahead of the latest tagged release.
Choose a model provider and enter your key when prompted. The local agent can
work with your files, run commands, and use registered services. It asks before
actions that change files or run commands. Reopen chat in the same directory
to continue; `/help` shows the controls.

## Build a capability, then use it

Describe what you need. For example, after installing the
[service-generation prerequisites](https://go-micro.dev/docs/quickstart.html#prerequisites):

```text
Build a service named notes that saves and lists notes. Store them persistently.
```

Review the proposed tool action. Once the service has built and registered, send
a new message to use its tools:

```text
Save a note: the launch review is on Friday. Then list my notes.
```

You now have Go source to inspect and extend, and service endpoints that other
applications can call too. The [complete walkthrough](https://go-micro.dev/docs/quickstart.html)
shows what to check, how to change the service, and how to run it after chat exits.
For a runnable example without a model key, see [first-agent](examples/first-agent/).

## Build on the framework

- **Services:** typed requests, discovery, RPC, and events. [Build a service](https://go-micro.dev/docs/client-server.html).
- **Agents:** models, history, tools, and approval policies you can extend in Go. [Build an agent](https://go-micro.dev/docs/guides/your-first-agent.html).
- **Workflows:** defined steps around service calls and agent decisions. [Compose a workflow](https://go-micro.dev/docs/guides/agents-and-workflows.html).

The CLI uses the same framework you import into your own programs. Infrastructure
components—including registry, transport, broker, and store—are replaceable Go
interfaces. Your services and agents can run independently of the development CLI.

[Documentation](https://go-micro.dev/docs/index.html) ·
[Go API](https://pkg.go.dev/go-micro.dev/v6) ·
[Examples](examples/README.md) ·
[Discord](https://discord.gg/G8Gk5j3uXr) ·
[Support](SUPPORT.md)

Apache 2.0 licensed.
