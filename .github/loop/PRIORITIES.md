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

1. **Add a first-agent wayfinding smoke test** ([#3879](https://github.com/micro/go-micro/issues/3879)) — #3870 closed via #3877, there are no open `codex` PRs in flight, and the North Star says the adoption on-ramp must be weighted at least as highly as internal hardening. The README now gives a clear no-secret → first-agent → debugging → 0→hero path, while the website/blog story says the product promise is one runtime for services, agents, and workflows. The next highest-value increment is to make that path a checked contract so docs wayfinding cannot silently drift from the lived harness story. Keep it local/no-secret and CI-verifiable by failing when the key guide links disappear or fall out of order.
2. **Add a no-secret inspect/debug transcript check** ([#3880](https://github.com/micro/go-micro/issues/3880)) — developer adoption is not just starting an agent; it is understanding what happened when the first conversation surprises you. Recent merged work strengthened streaming memory, run-event traces, preflight failures, and the plan/delegate handoff, so the next DX seam is proving the inner loop from `micro chat` to inspect/history with a mock provider. This is a Now-phase getting-started contract item and should point failures back to the debugging guide rather than redesigning the CLI.
3. **Add an AP2 mandate layer over A2A and x402** ([#3552](https://github.com/micro/go-micro/issues/3552)) — this remains a forward interop investment, not a Now/Next blocker: Go Micro already has A2A agents and x402 paid tools, so a small signed-mandate foundation can keep agent payments aligned with the open-protocol story without pulling the queue ahead of adoption, resilience, streaming, or durable agent operation. Keep it additive and opt-in while the AP2/FIDO work settles.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
