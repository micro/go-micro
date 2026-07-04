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

1. **Add deadline propagation and bounded retry around provider calls** ([#3891](https://github.com/micro/go-micro/issues/3891)) — #3889 shipped the no-secret chat → inspect/history checkpoint, so the adoption on-ramp now has install, scaffold, run, chat, inspect, and 0→hero coverage and #3880 is closed. With no open `codex` PRs in flight, the highest-value Now-phase gap moves to resilience: real first agents still need cancellation, timeout, rate-limit, and transient-provider failures to be deterministic, actionable, and safe from duplicate completed tool execution. Keep this additive and CI-verifiable; do not redesign the public provider API without a human.
2. **Add an AP2 mandate layer over A2A and x402** ([#3552](https://github.com/micro/go-micro/issues/3552)) — this remains a forward interop investment, not a Now/Next blocker: Go Micro already has A2A agents and x402 paid tools, so a small signed-mandate foundation can keep agent payments aligned with the open-protocol story without pulling the queue ahead of adoption, resilience, streaming, or durable agent operation. Keep it additive and opt-in while the AP2/FIDO work settles.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
