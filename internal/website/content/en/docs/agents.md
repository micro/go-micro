---
title: "Agents"
params:
  docs_main: true
  exclude_search: false
---

An agent combines a model, instructions, and tools. Service endpoints become its
tools, so you can use the same operations from code and conversation.

## Create an agent

Keep the [notes service](services.md) running. In your project, create
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
micro chat
```

With `assistant` as the only registered agent, chat connects to it. Ask:

```text
What notes do I have?
```

The agent can call `Notes.List` and answer using its result. Check the tool
activity to see that the service was called. This uses your model provider and
incurs its normal API charges.

Use `/agents` to check the connection. If other agents are running,
`micro chat assistant` connects to this one explicitly.

Your Go code controls the provider, instructions, and allowed services. Change
those options to build your own agent. For programmatic calls, the agent exposes
`Ask(ctx, message)`; see the [agent API](https://pkg.go.dev/go-micro.dev/v6/agent).

To use Mistral with this example, change the provider to `"mistral"` and
the key variable to `"MISTRAL_API_KEY"`. The service and tools stay the same.
Set the key in the terminal running the agent, not the chat terminal.

**Next: [coordinate work with a workflow →](workflows.md)**
