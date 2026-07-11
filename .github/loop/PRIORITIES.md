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

1. **Fix race in zero-to-hero CLI output polling test** ([#4658](https://github.com/micro/go-micro/issues/4658)) — The first-agent chat/inspect fixture shipped, but CI immediately exposed an unsynchronized `strings.Builder` read/write in the new harness under `-race`. Put this first because the getting-started contract is only credible if the provider-free chat/inspect proof is green in the same race-enabled CI that protects every increment.
2. **Add scheduled cross-provider agent conformance** ([#4649](https://github.com/micro/go-micro/issues/4649)) — Once the no-secret first-agent fixture is race-safe, the Now roadmap still calls for battle-testing the same agent scenario across providers. Verify tool-calling, multi-step turns, plan/delegate, and guardrails on the live-provider matrix behind key-gated scheduled CI, without slowing the local harness.
3. **Harden agent provider failure resilience** ([#4650](https://github.com/micro/go-micro/issues/4650)) — The next reliability seam is making timeouts, cancellation, rate limits, retry/backoff, and inspectable failure metadata predictable through the agent loop. Keep this scoped to existing behavior and tests; surface any breaking API or default changes as human review notes instead of queue work.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
