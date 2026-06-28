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

## Next (ranked)

1. **Add durable checkpoint/resume support to agent runs** ([#3254](https://github.com/micro/go-micro/issues/3254)) — with both 0→1 and 0→hero now CI-verified, make long-running agents as resumable as flows so the harness can safely run scheduled and looping work without replaying completed tool side effects.
2. **Broaden provider-backed AI streaming conformance** ([#3255](https://github.com/micro/go-micro/issues/3255)) — keep streaming coherent from provider adapters through chat and A2A by testing chunks, completion, cancellation, and error propagation across supported providers.
3. **Export agent RunInfo as OpenTelemetry spans** ([#3256](https://github.com/micro/go-micro/issues/3256)) — connect the existing run/model/tool metadata to standard tracing so developers can inspect agent behavior with the same operational tools they use for services.

## Later (ranked)

4. **Add A2A resubscribe and input-required handoff support** ([#3235](https://github.com/micro/go-micro/issues/3235)) — after push notifications and multi-turn continuation shipped, finish the remaining long-running A2A interoperability gap: reconnecting to live task streams and carrying human-input-required handoffs through the gateway.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
