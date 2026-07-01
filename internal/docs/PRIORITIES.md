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

1. **Add a “Debugging your agent” guide focused on the dev workflow** ([#3564](https://github.com/micro/go-micro/issues/3564)) — the inner loop is scaffold → run → chat → inspect → deploy. Document how to see tool calls, run history, provider failures, guardrail refusals, and flow handoffs before adding more depth that users cannot diagnose.
2. **Finalize the universe notify step after an agent-backed timeout** ([#3589](https://github.com/micro/go-micro/issues/3589)) — recent live-provider CI exposed a real services → agents → workflows seam: an agent-backed notify side effect can complete while the client observes a timeout, leaving pending flow state and late duplicate notifications. Keep this high because it protects the 0→hero/reliability contract, but leave the adoption guide first so the queue does not drift back to only internal hardening.
3. **Prevent duplicate tool side effects in the plan/delegate harness** ([#3559](https://github.com/micro/go-micro/issues/3559)) — correctness still matters where it protects real user trust. Plan/delegate is central to the services → agents lifecycle, and duplicate side effects undermine the “agent as dependable service” story.
4. **Expose `fallback_echo` during A2A streaming fallback conformance** ([#3560](https://github.com/micro/go-micro/issues/3560)) — keep interop conformance trustworthy without letting it dominate the adoption queue. This is scoped, testable, and protects the A2A promise developers see in the README and site.
5. **Propagate agent run cancellation and deadlines through model and tool calls** ([#3544](https://github.com/micro/go-micro/issues/3544)) — after the on-ramp items, the highest-value remaining Now-phase resilience gap is predictable failure semantics across agent runs, model calls, tool calls, plan/delegate, and flow handoffs. Tool retries and live-provider deadline tuning are in place; the lifecycle still needs cancellation/deadline propagation so work fails safely instead of becoming opaque loops.
6. **Emit OpenTelemetry spans for agent run timelines** ([#3525](https://github.com/micro/go-micro/issues/3525)) — recent work made runs inspectable, correlated trace metadata through scheduled dispatch, verified restart resume, added opt-in tool retries, hardened provider conformance, and fixed provider-emitted text tool calls. The next Next-phase step is to turn that RunInfo foundation into standard OTel spans for agent runs, model calls, tool calls, checkpoint/resume, cancellation/deadlines, and failures.
7. **Add an AP2 mandate layer over A2A and x402** ([#3552](https://github.com/micro/go-micro/issues/3552)) — this is a forward interop investment, not a Now-phase blocker: Go Micro already has A2A agents and x402 paid tools, so a small signed-mandate foundation can keep agent payments aligned with the open-protocol story without pulling the queue away from adoption, resilience, or observability. Keep it additive and opt-in while the AP2/FIDO work settles.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
