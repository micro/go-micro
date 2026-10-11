# TypeSafe

A decision provider for the [TypeSafe System One API](https://docs.typesafe.ai/api).
Uses Jev to evaluate named questions against text or structured data.

```go
package main

import (
    "context"
    "fmt"
    "os"

    "go-micro.dev/v6/model"
    _ "go-micro.dev/v6/model/typesafe"
)

func main() {
    decider := model.NewDecider("typesafe",
        model.WithAPIKey(os.Getenv("TYPESAFE_API_KEY")),
    )

    result, err := decider.Decide(context.Background(), &model.DecisionRequest{
        State: "My payouts have been failing for three days.",
        Questions: map[string]model.Question{
            "urgent": {
                Type:         model.Noul,
                Instructions: "Does this need urgent attention?",
            },
        },
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(result.Answers["urgent"].Noul)
}
```

`Noul` returns the probability of yes, from 0 to 1. `Choice` returns an option
name; its criteria are a map of names to descriptions. `Score` returns a weighted
level index; its criteria are an ordered array of 2–10 descriptions.

```go
questions := map[string]model.Question{
    "team": {
        Type:         model.Choice,
        Instructions: "Which team should handle this?",
        Criteria: map[string]string{
            "billing": "Payments and invoices",
            "support": "Product issues",
        },
    },
    "severity": {
        Type:         model.Score,
        Instructions: "How severe is the issue?",
        Criteria:     []string{"Minor", "Disruptive", "Blocking"},
    },
}
```

Read the matching answer's `Choice` or `Score` field. Both include
`Probabilities` and `Confidence`; scores also include a `Legend`.
Confidence summarizes the distribution, not a guarantee of correctness.
The application decides what to do with each result.

The default model is `jev-latest`. Use `model.WithModel` to change it or
`DecisionRequest.Model` to override it for one request. `model.WithBaseURL`
overrides `https://api.typesafe.ai`; the provider appends `/v1/systemone`.
State and instructions accept text, objects, or arrays. Criteria descriptions
may also contain structured data; use `map[string]any` or `[]any` when needed.

Requests use the supplied context for cancellation and deadlines. Set a deadline
for bounded calls. HTTP failures return `model.HTTPError`, preserving status and
retry information. There are no automatic retries. Invalid or incomplete answers
return an error rather than a successful zero-valued decision.

This provider implements `model.Decider`, not `model.Model`. Call it from normal
Go code, a service handler, or a flow step; it is not a chat provider.
