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

1. **Broaden provider-backed agent streaming end to end** ([#3857](https://github.com/micro/go-micro/issues/3857)) — #3861 closed the durable agent resume discoverability gap, and there are no open `codex` PRs in flight. Streaming is now the highest-value remaining Next-phase increment because the README, website, and roadmap all sell an interactive services → agents → workflows lifecycle; responsive `micro chat`, provider-backed `ai.Stream`, and A2A long-task UX are part of the adoption on-ramp, not just internal depth. Keep this scoped to additive provider-backed coverage, mock/live-provider tests, and discoverable support status so developers can trust the chat/agent/A2A path without a public-API rewrite.
2. **Add an AP2 mandate layer over A2A and x402** ([#3552](https://github.com/micro/go-micro/issues/3552)) — this remains a forward interop investment, not a Now/Next blocker: Go Micro already has A2A agents and x402 paid tools, so a small signed-mandate foundation can keep agent payments aligned with the open-protocol story without pulling the queue ahead of adoption, resilience, streaming, or durable agent operation. Keep it additive and opt-in while the AP2/FIDO work settles.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
