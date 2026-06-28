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

1. **Add cross-provider agent conformance checks** ([#3240](https://github.com/micro/go-micro/issues/3240)) — the resilience seam closed, so the next highest-value hardening gap is proving one representative agent/tool scenario across providers with key-gated CI/scheduled checks. This protects the harness promise that agents behave consistently on one runtime even as provider adapters evolve.
2. **CI-verify the 0-to-1 getting-started path** ([#3234](https://github.com/micro/go-micro/issues/3234)) — add a deterministic no-key check for scaffold → run/build → call so the README/blog promise that building a service stays effortless cannot regress while the agent stack deepens.
3. **CI-verify the 0-to-hero agent workflow** ([#3241](https://github.com/micro/go-micro/issues/3241)) — after the basic service contract is guarded, make the full services → agents → workflows story executable in CI with a maintained no-secret reference scenario for run → chat → inspect boundaries.

## Later (ranked)

4. **Add A2A resubscribe and input-required handoff support** ([#3235](https://github.com/micro/go-micro/issues/3235)) — after push notifications and multi-turn continuation shipped, finish the remaining long-running A2A interoperability gap: reconnecting to live task streams and carrying human-input-required handoffs through the gateway.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
