# Priorities

The ranked work queue for the autonomous improvement loop. The
**architecture-review** pass (the *architect*) owns this file: each run it turns
the [roadmap](../../ROADMAP.md) plus an internal scan (gaps in the
services → agents → workflows lifecycle, API coherence, drift, tech debt, test and
DX friction) into a single ordered list — highest-value first — and links each
item to a tracking issue. The hourly **continuous-improvement** pass works the
**top item whose issue is still open**. So the architect decides *what*, and the
increment loop *builds* it.

**Reading / editing.** An item is done when its linked issue closes (the increment
that builds it adds `Closes #<issue>`). Roadmap phase (Now → Next → Later) is the
primary ordering; internal findings are interleaved by value, not kept in a
separate list. The human can reorder this list — or the issues — at any time to
redirect the loop; direction always wins.

**Off-limits to the loop** (the architect proposes these as notes, never as queue
items the loop can auto-merge): brand/positioning copy, breaking public-API
changes, architectural rewrites. Those go to the human.

## Work queue (ranked)

1. **Update AI capability tests for minimax provider registration** ([#3787](https://github.com/micro/go-micro/issues/3787)) — the latest master `Run Tests` workflow is red because the AI provider capability fixtures still omit the newly registered `minimax` provider. This is the top Now-phase item because the autonomous loop's evaluator is blocked by a deterministic unit-test regression, and the fix is narrow, CI-verifiable, and directly protects cross-provider adoption work.
2. **Fix atlascloud plan-delegate harness missing delegated notify side effect** ([#3760](https://github.com/micro/go-micro/issues/3760)) — PR #3783 closed the buyer-recipient universe blocker (#3772), leaving plan/delegate as the live atlascloud/minimax 0→hero services → agents → workflows failure. Keep this ahead of broader hardening until the conductor → delegated comms → notify path reliably proves the multi-agent side effect developers see in the harness.
3. **Isolate file-store tests from shared default directory** ([#3751](https://github.com/micro/go-micro/issues/3751)) — repeated `go test -race -cover ./...` failures still point at file-store reliability under the shared default directory/table setup. This remains the highest-value store/CI stability issue because a flaky evaluator erodes the loop's ability to ship adoption work safely.
4. **Stabilize file-store suffix expiry test timing** ([#3780](https://github.com/micro/go-micro/issues/3780)) — the newer store failure is a narrower timing-sensitive suffix-expiry assertion under `-race -cover`. Keep it adjacent to #3751 but separate because it may need a focused TTL/assertion fix even if directory isolation improves the broader file-store tests.
5. **Propagate agent run cancellation and deadlines through model and tool calls** ([#3544](https://github.com/micro/go-micro/issues/3544)) — once the live harness and red CI blockers are cleared, the highest-value remaining resilience gap is predictable failure semantics across agent runs, model calls, tool calls, plan/delegate, and flow handoffs. Tool retries, live-provider deadline tuning, delegated-plan completion, and side-effect enforcement are in place; the lifecycle still needs cancellation/deadline propagation so work fails safely instead of becoming opaque loops.
6. **Emit OpenTelemetry spans for agent run timelines** ([#3525](https://github.com/micro/go-micro/issues/3525)) — recent work made runs inspectable, correlated trace metadata through scheduled dispatch, verified restart resume, added opt-in tool retries, hardened provider conformance, and fixed provider-emitted text tool calls. The next Next-phase step is to turn that RunInfo foundation into standard OTel spans for agent runs, model calls, tool calls, checkpoint/resume, cancellation/deadlines, and failures.
7. **Add an AP2 mandate layer over A2A and x402** ([#3552](https://github.com/micro/go-micro/issues/3552)) — this is a forward interop investment, not a Now-phase blocker: Go Micro already has A2A agents and x402 paid tools, so a small signed-mandate foundation can keep agent payments aligned with the open-protocol story without pulling the queue away from adoption, resilience, or observability. Keep it additive and opt-in while the AP2/FIDO work settles.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
