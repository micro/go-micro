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

1. **Add CI-verified first-agent chat/inspect fixture** ([#4644](https://github.com/micro/go-micro/issues/4644)) — With plan/delegate persistence hardened and the first-agent breadcrumbs refreshed, the highest-value remaining adoption gap is proving the real CLI-shaped scaffold → run → chat → inspect loop for the smallest provider-free agent. Keep this focused and separate from the richer support 0→hero reference so a new developer can trust the first-agent path without provider keys.
2. **Add scheduled cross-provider agent conformance** ([#4649](https://github.com/micro/go-micro/issues/4649)) — The Now roadmap still calls for battle-testing the same agent scenario across providers. After the no-secret first-agent fixture, verify tool-calling, multi-step turns, plan/delegate, and guardrails on the live-provider matrix behind key-gated scheduled CI, without slowing the local harness.
3. **Harden agent provider failure resilience** ([#4650](https://github.com/micro/go-micro/issues/4650)) — The next reliability seam is making timeouts, cancellation, rate limits, retry/backoff, and inspectable failure metadata predictable through the agent loop. Keep this scoped to existing behavior and tests; surface any breaking API or default changes as human review notes instead of queue work.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
