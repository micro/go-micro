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

## Now (ranked)

1. **Harden agent model-call resilience** ([#3233](https://github.com/micro/go-micro/issues/3233)) — complete the remaining hardening seam from the roadmap: deadline/cancellation propagation, bounded retry/backoff, and inspectable timeout/rate-limit/cancellation outcomes for the agent loop. This is the highest-value next item because durable runs, streaming, observability, memory, HITL, x402, and A2A continuation now exist; the shared operational harness still needs to fail safely under real provider conditions.
2. **CI-verify the 0-to-1 getting-started path** ([#3234](https://github.com/micro/go-micro/issues/3234)) — add a deterministic no-key check for scaffold → run/build → call so the README/blog promise that building a service stays effortless cannot regress while the agent stack deepens.

## Later (ranked)

3. **Add A2A resubscribe and input-required handoff support** ([#3235](https://github.com/micro/go-micro/issues/3235)) — after push notifications and multi-turn continuation shipped, finish the remaining long-running A2A interoperability gap: reconnecting to live task streams and carrying human-input-required handoffs through the gateway.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
