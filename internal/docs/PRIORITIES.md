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

## Later (ranked)

1. **x402 spend caps and live facilitator conformance** (#3211) — enforce paid-tool
   budgets and add credential-gated live facilitator checks. Roadmap → x402 paid
   remote tools; ranked first because memory compaction and human-in-the-loop
   pause/resume have shipped, leaving paid-tool trust as the highest-value
   operability seam before agents perform longer-running or paid work.
2. **A2A push notifications and multi-turn task support** (#3212) — extend the A2A
   gateway/client path beyond streaming lifecycle updates into push and multi-turn
   task state. Roadmap → A2A push notifications and multi-turn tasks; ranked after
   x402 hardening because interop depth compounds best once local run and paid-tool
   execution are bounded, inspectable, and budgeted.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
