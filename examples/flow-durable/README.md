# Durable Flow

A workflow that survives a crash and resumes where it stopped.

A `flow` can be an ordered list of **steps** — a task with stages —
instead of a single LLM turn. Each step is checkpointed before and after
through a pluggable `Checkpoint` (store-backed by default), so if the
process dies mid-run, the run resumes at the step it stopped on, without
re-running the steps that already completed (and already had their side
effects).

## What this shows

A three-step checkout (`reserve → charge → confirm`) whose `charge` step
fails the first time, simulating a transient outage / crash:

```
first run:
  reserve  → inventory reserved
  charge   → payment dependency unavailable (crash)
  run failed: payment gateway timeout

checkpoint: run 70643f61 is at step "charge" (status failed)

resume:
  charge   → payment captured
  confirm  → order confirmed

reserve ran 1 time(s) total — completed steps are not repeated on resume
no pending runs — the workflow completed durably
```

The key line is the last pair: on `Resume`, `reserve` does **not** run
again — its result was checkpointed — and the run finishes.

## The pieces

```go
f := micro.NewFlow("checkout",
    micro.FlowSteps(
        micro.FlowStep{Name: "reserve", Run: reserve},
        micro.FlowStep{Name: "charge",  Run: charge},
        micro.FlowStep{Name: "confirm", Run: confirm},
    ),
    micro.FlowWithCheckpoint(micro.StoreCheckpoint(nil, "checkout")), // nil store = default; "checkout" = key scope
)

f.Execute(ctx, `{}`)        // runs; crashes at charge
pending, _ := f.Pending(ctx) // the run, checkpointed at "charge"
f.Resume(ctx, pending[0].ID) // continues from charge to the end
```

- **`State`** carries a typed payload (`Set`/`Scan`) plus a `Stage`
  marker — the resume point.
- **`Checkpoint`** persists each `Run`. The built-in is store-backed and
  namespaces keys by flow name (`flow/checkout/runs/...`), so one flow's
  runs don't share a keyspace with another's. Point the default store at
  Postgres or NATS KV and a run survives a real process restart, or
  implement the interface to plug in Temporal, Restate, etc.
- A real step would be `flow.Call(service, endpoint)` (an RPC),
  `flow.Dispatch(agent)` (hand off to an agent), or `flow.LLM(prompt)`
  (one model turn). Here they're plain funcs so durability is the only
  thing on display.

## Run

```bash
go run main.go
```

No LLM key required.
