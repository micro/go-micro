# Services, agents, and flows across restarts

A flow prepares work through an RPC service, asks an agent to review it using a
service tool, waits for approval, and commits through the service. The default
model is deterministic and makes no external requests.

From the repository root, run these as separate processes:

```sh
go run ./examples/durable-flow -action start
go run ./examples/durable-flow -action status
go run ./examples/durable-flow -action approve
go run ./examples/durable-flow -action start
```

The first process exits with `status=waiting`. The second reads the persisted
run, including step attempts and the child agent run ID. Approval advances the
same run to `done`. Repeating `start` with the same ID returns the completed run;
it does not repeat the model or service effects. Use `-id another-run` for new
work. `resume` recovers interrupted work but cannot approve a waiting run.

`-dir` selects the journal directory (default `.durable-flow`). `runs.db` holds
execution records and `effects.db` holds the service's effects. The service
atomically stores each effect under the stable `micro-idempotency-key` carried
by its flow call. This is application-level deduplication, not an exactly-once
promise for arbitrary external systems.

The journal supports one owning process at a time. It closes between commands,
so each command proves recovery with a new service, flow, agent, and client.
Discovery and broker state are in memory; service calls use local RPC. No Mu
runtime, multicast discovery, or web UI is required.

For an optional real model, export `MICRO_AI_API_KEY` and pass `-provider openai`
(or another built-in provider) and optionally `-model`. `MICRO_AI_BASE_URL` sets
a custom endpoint. Use the same provider/model when resuming a run. Real models
are not part of the default acceptance test and can incur provider charges.

```sh
go test -race ./examples/durable-flow
```

The framework recovery tests also cover duplicate admission, concurrent owners,
ambiguous interrupted actions, cancellation, and consumed retry budgets. A
non-idempotent action interrupted before its result is saved stops for
reconciliation. Whole-loop replay is rejected in strict mode. Remote dispatch requires a
registered agent advertising fenced stable-run support.
