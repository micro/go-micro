---
title: "Build your first project"
linkTitle: "Quick Start"
description: "Start a conversation, build a service, use it, and keep developing the Go code."
---

Go Micro lets you build a capability as a service and give an agent tools to use
it. This walkthrough starts in an empty directory and ends with source code you
can change and a service you can run independently.

## 1. Open a project

Install [Go 1.26+](https://go.dev/doc/install), then the development CLI:

```sh
go install go-micro.dev/v6/cmd/micro@master
export PATH="$(go env GOPATH)/bin:$PATH"
mkdir my-app && cd my-app
micro chat
```

The chat experience described here is on `master`, ahead of the latest tagged
release. `@latest` and the binary installer install the tagged release instead.

Choose your model provider and enter a key. An existing provider environment key
can skip setup. Use `/provider` to change provider, `/model` to choose a model that
supports tool calling, or `/login` to replace a key. These settings and your
conversation are saved for this project.

With no registered agents, you are talking to the local development agent. It can
read your project and propose file changes, commands, and service calls.
`/approve` allows the displayed action; `/deny` refuses it. `/stop` cancels the
current request. `/help` lists the controls.

If another agent is already registered, chat connects to it instead. `/agents`
shows where your requests are going. Stop that host to follow this local
walkthrough; see [connected agents](interactive-agent.md) for working with hosts.

## 2. Build a capability

### Prerequisites

The service generator uses Protocol Buffers. Before asking it to generate a
service, install `make` and [protoc](https://protobuf.dev/installation/) through
your package manager, then install the Go plugins in another terminal:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install go-micro.dev/v6/cmd/protoc-gen-micro@master
export PATH="$(go env GOPATH)/bin:$PATH"
protoc --version
command -v make protoc-gen-go protoc-gen-micro
```

All four tools must be on the PATH used to launch `micro chat`. Restart chat if
you changed PATH. These tools are needed for service generation, not ordinary
file editing and command execution.

### Describe what you need

In chat, ask for a small capability:

```text
Build a service named notes that saves and lists notes. Store them persistently.
```

Review the generation action before approving it. The generator asks your model
for a service design and implementation, writes Go and protobuf source, and
attempts to build and start the service. Generation makes additional model calls;
its output and timing depend on your provider and model.

**Check the result:** wait for successful build and service registration. Do not
continue on a compilation or registration error. Read the error, correct the
missing dependency or generated code, and retry. Files already written remain in
your project.

Each generated service has its own `go.mod`, `main.go`, `handler/`, and `proto/`.
Inspect these files: this is the implementation you own and can extend.

## 3. Use the service

After the service registers, send a **new message** so chat discovers its tools:

```text
Save a note: the launch review is on Friday. Then list my notes.
```

Approve the service action after checking its arguments. You should see tool
activity for saving and listing, with your note in the result. A conversational
answer alone is not evidence that a note was saved.

You can also call the service directly from another terminal:

```sh
micro call notes Notes.List '{}'
```

That command assumes the generated service is named `notes` with a `Notes.List`
endpoint. Check the generated `.proto` file if the model chose different names.
The list should contain the saved note. This is the same service capability an
application or another agent can call.

## 4. Extend the implementation

Ask the agent to read the existing code before changing it:

```text
Read the notes service. Add search, rebuild it, and verify that searching for
launch returns the note we saved.
```

Review the edits and commands. Ask it to show the verification result, and inspect
the changes in your editor. You can edit the Go code yourself as well. The
framework handles communication; your handlers define what the service does.

## 5. Run it after chat exits

`/exit` stops services launched by this chat process. It does not remove their
source code. To run the generated notes service independently, open a terminal
in your project and run:

```sh
cd notes
go run .
```

Leave it running. In another terminal, return to the original `my-app` directory
and run `micro chat` again. Your conversation resumes and the service is available
as tools. Ask it to list the notes again to check that the generated implementation
really persisted the data.

For hot reload, a gateway, and running multiple services, continue with
[local development](getting-started/index.md). When you want an agent with
its own instructions and a selected set of services, follow
[Your First Agent](guides/your-first-agent.md). Use
[workflows](guides/agents-and-workflows.md) for repeatable sequences of work.

## Develop without a model

Services do not require AI. [Client/server](client-server.md) covers building and
calling services directly in Go. The runnable
[first-agent example](https://github.com/micro/go-micro/tree/master/examples/first-agent)
uses a scripted model to demonstrate a real service call without a provider key.
That example verifies framework wiring, not live model behaviour.
