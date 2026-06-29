# Agent Human Input Pause/Resume

Agents can pause a durable run when the model needs a human decision before it
can continue. This keeps the services → agents → workflows lifecycle in one
runtime: services expose tools, the agent decides it needs operator input, and
the same checkpointed run resumes once that input arrives.

## Pattern

```go
cp := flow.StoreCheckpoint(nil, "deploy-agent")
ag := agent.New(
    agent.Name("deploy-agent"),
    agent.WithCheckpoint(cp),
)

resp, err := ag.Ask(ctx, "Deploy the service")
if err != nil {
    // If the model called the built-in request_input tool, the run is saved as
    // paused/input-required instead of losing state or completing early.
    pending, _ := agent.Pending(ctx, ag)
    runID := pending[0].ID

    // Later, after an operator supplies the missing answer, the same run ID
    // continues with the original prompt, human input, memory, and completed
    // tool history intact.
    resp, err = agent.ResumeInput(ctx, ag, runID, "Deploy to us-east-1")
}
_ = resp
```

The model sees a built-in `request_input` tool with a `prompt` argument. When it
calls that tool, Go Micro persists the run with status `paused` and stage
`input-required`. Plain `agent.Resume` continues to support completed, failed,
and approval-paused runs; input-required runs are resumed with
`agent.ResumeInput` so the human response is explicit.

## Cancellation and deadlines

`ResumeInput` uses the caller's `context.Context` for checkpoint reads, writes,
and the resumed model/tool turn. If the context is canceled or its deadline
expires before the resume is committed, the call returns the context error and
the checkpointed run remains `paused` at `input-required`; list it with
`agent.Pending` and retry with a fresh context once the operator is ready.
