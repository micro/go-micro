---
title: "Agent Loops"
---

Most agent work is one-shot: a prompt goes in, an answer comes out. The next
step in agentic systems is the **loop** — run a step over and over, letting the
agent keep working until the goal is met instead of stopping after one pass. One
agent improves an architecture while another removes duplicated abstractions,
both opening pull requests continuously; a draft is refined until it's good
enough; a build is fixed and re-run until it's green.

The catch is cost and runaway risk: a loop "burns through tokens a lot faster
than a simple Q&A chatbot," and a non-deterministic stop ("keep going until
you're done") has no natural ceiling. So a usable loop needs two things:

1. a **stop condition** — how it decides it's done, and
2. a **hard cap** — a guardrail that guarantees it always terminates.

Go Micro gives you both as a flow step: `micro.FlowLoop`.

## The shape

`micro.FlowLoop` is a `StepFunc`, so it drops into a flow's ordered, checkpointed
step list like any other step. It runs a **body** step repeatedly, carrying the
flow `State` from one pass to the next, until a stop condition fires or the
iteration cap is hit — whichever comes first.

```go
f := micro.NewFlow("refactor",
    micro.FlowProvider("anthropic"),
    micro.FlowSteps(
        micro.FlowStep{Name: "improve", Run: micro.FlowLoop(
            micro.FlowDispatch("coder"), // the body: an agent does one pass
            micro.FlowUntilLLM("Is the refactor complete with no duplicated abstractions left?"),
            micro.FlowLoopMax(5),        // the ceiling: never more than 5 passes
        )},
    ),
)
```

## Stop conditions

**Code-defined** — `FlowUntil` stops when your predicate returns true. Use it
when "done" is something you can measure (tests pass, a score clears a
threshold, a queue is empty):

```go
micro.FlowUntil(func(_ context.Context, s micro.FlowState, iter int) (bool, error) {
    var d Draft
    _ = s.Scan(&d)
    return d.Quality >= 90, nil
})
```

**Model-judged** — `FlowUntilLLM` asks the flow's model, after each pass,
whether the goal is met, and stops on an affirmative answer. This is the
supervised ("Ralph") loop: the agent decides when it's done, while the cap
still guarantees it stops. It requires a flow model (`FlowProvider`/`FlowAPIKey`).

```go
micro.FlowUntilLLM("Have all the failing tests been fixed?")
```

You can combine both — either firing stops the loop.

## The guardrail

`FlowLoopMax(n)` is the ceiling. The body never runs more than `n` times, so the
loop always terminates even if the stop condition never fires. When the cap is
hit, the loop returns the latest state rather than erroring — the guardrail did
its job. **Always set it.** For tighter budgets, keep the cap low and pair the
loop with [agent guardrails](agent-guardrails.md) (e.g. token/spend limits)
and [paid tools](x402-payments.md) (per-call metering) so a background loop
can't run up an unbounded bill.

## Watching progress

`FlowOnIteration` runs after each pass — log it, or persist a summary so you can
see how a long-running loop is doing:

```go
micro.FlowOnIteration(func(iter int, s micro.FlowState) {
    log.Printf("pass %d: %s", iter, s.String())
})
```

## Durability

A loop runs as a **single flow step**. The flow checkpoints the loop's outcome
(before and after the step) through its [Checkpoint](../deployment.md), and a
resume re-enters the step — so keep loop bodies safe to repeat. For long loops,
use `FlowOnIteration` to persist per-pass progress.

## Run it

A complete, offline example (no API key — the body and stop condition are plain
Go) is in [`examples/flow-loop`](https://github.com/micro/go-micro/tree/master/examples/flow-loop):

```bash
go run ./examples/flow-loop/
# refining until quality >= 90
#   pass 1 → quality 30
#   pass 2 → quality 60
#   pass 3 → quality 90
# done: {"text":"draft refined (quality 90)","quality":90}
```

Swap the body for `micro.FlowDispatch("agent")` or `micro.FlowLLM(...)` and the
stop check for `micro.FlowUntilLLM(...)` to turn it into a real agent loop.

## See also

- [Agents and Workflows](agents-and-workflows.md) — flows vs. agents
- [Agent Guardrails](agent-guardrails.md) — bounding what a loop can do
- [Plan & Delegate](plan-delegate.md) — splitting work across agents
