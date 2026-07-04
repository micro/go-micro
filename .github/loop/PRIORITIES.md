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

1. **Trace agent `RunInfo` into OpenTelemetry spans** ([#3867](https://github.com/micro/go-micro/issues/3867)) — #3857/#3865 closed the streaming queue item, there are no open `codex` PRs in flight, and the remaining Next-phase gap with the strongest mission fit is operability of the services → agents → workflows lifecycle. The README, website, roadmap, and recent blog all promise that agents are services you can run, chat with, inspect, and trust under the same runtime; projecting run steps, tool calls, delegation, streaming, cancellation, and errors into OTel spans makes that lived path debuggable without a public-API rewrite. Keep this additive/no-op when tracing is disabled, with tests proving representative `RunInfo` events become trace data.
2. **Add an AP2 mandate layer over A2A and x402** ([#3552](https://github.com/micro/go-micro/issues/3552)) — this remains a forward interop investment, not a Now/Next blocker: Go Micro already has A2A agents and x402 paid tools, so a small signed-mandate foundation can keep agent payments aligned with the open-protocol story without pulling the queue ahead of adoption, resilience, streaming, or durable agent operation. Keep it additive and opt-in while the AP2/FIDO work settles.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
