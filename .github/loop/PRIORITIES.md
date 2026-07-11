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

1. **Fix in-memory stream Nack redelivery ordering** ([#4690](https://github.com/micro/go-micro/issues/4690)) — Master is red from `events` `TestStream/memory/AckingNacking`, where manual NACK can let unrelated queued messages arrive before the nacked event is redelivered. The loop depends on green CI as its evaluator, and reliable event redelivery underpins flows and agent-triggered workflows, so restore the existing contract before deeper feature work.
2. **Harden agent provider failure resilience** ([#4650](https://github.com/micro/go-micro/issues/4650)) — With scheduled cross-provider conformance closed by #4678, the first-agent fixture races closed by #4684/#4667, and initial provider failure metadata recorded by #4688, the next Now-roadmap reliability seam is making timeouts, cancellation, rate limits, retry/backoff, and inspectable failure metadata predictable through the agent loop. Keep this scoped to existing behavior and tests; surface any breaking API or default changes as human review notes instead of queue work.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
