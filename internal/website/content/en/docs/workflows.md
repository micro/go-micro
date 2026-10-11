---
title: "Workflows"
params:
  docs_main: true
  exclude_search: false
---

A workflow runs named steps in order. Each step receives the previous step's
result. Use a workflow when you know the sequence; use an agent when a model
needs to choose what to do next.

## Run two steps

Keep the [notes service](services.md) running. Create `cmd/workflow/main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "go-micro.dev/v6/flow"
)

func main() {
    f := flow.New("read-notes", flow.Steps(
        flow.Step{Name: "list", Run: flow.Call("notes", "Notes.List")},
        flow.Step{Name: "print", Run: func(ctx context.Context, state flow.State) (flow.State, error) {
            fmt.Println(state.String())
            return state, nil
        }},
    ))
    if err := f.Execute(context.Background(), "{}"); err != nil {
        log.Fatal(err)
    }
}
```

Run it from your project:

```sh
go run ./cmd/workflow
```

The first step calls the service. The second prints its JSON response. A step
can call a service, dispatch work to an agent, or run your own Go function.

## Retries and saved progress

Flows save progress between steps. The default checkpoint uses a file store;
retain that data if you need to resume after a restart. Completed steps are
skipped on resume, but an interrupted step can run again. Make operations safe
to repeat before enabling retries.

For configuration and recovery methods, see the
[flow API](https://pkg.go.dev/go-micro.dev/v6/flow).
