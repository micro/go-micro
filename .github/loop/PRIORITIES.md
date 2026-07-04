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

1. **Fix plan-delegate completing before comms notification** ([#3870](https://github.com/micro/go-micro/issues/3870)) — #3867 closed via #3872, there are no open `codex` PRs in flight, and the only fresh open `codex` Now-phase issue is a live provider harness failure where the conductor creates the required task side effects but completes before the delegated notify side effect. This is the highest-value next increment because the roadmap makes cross-provider conformance, failure/resilience, and the getting-started/0→hero contract the current gate; the README and website now promise a walkable scaffold → run → chat → inspect lifecycle, and a flaky plan/delegate handoff undermines trust in services → agents → workflows more than another forward-looking interop layer. Keep the fix scoped and CI-verifiable with a regression proving `notify=1` and no duplicate task/notification side effects.
2. **Add an AP2 mandate layer over A2A and x402** ([#3552](https://github.com/micro/go-micro/issues/3552)) — this remains a forward interop investment, not a Now/Next blocker: Go Micro already has A2A agents and x402 paid tools, so a small signed-mandate foundation can keep agent payments aligned with the open-protocol story without pulling the queue ahead of adoption, resilience, streaming, or durable agent operation. Keep it additive and opt-in while the AP2/FIDO work settles.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
