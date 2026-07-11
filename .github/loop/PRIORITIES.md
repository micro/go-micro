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

1. **Verify first-agent docs commands in CI** ([#4696](https://github.com/micro/go-micro/issues/4696)) — With #4702 closing the broad provider retry-cancellation seam, the highest-value Now-roadmap work returns to developer adoption: make the README/website first-agent path a focused no-secret command contract. The mission is to make services → agents → workflows feel like one runtime, and the current risk is not lack of depth but an on-ramp that can drift from runnable behavior. Keep this scoped to verified commands, example wayfinding, and short mock-model checks; do not redesign public APIs.
2. **Harden AtlasCloud plan-delegate tool-call recovery** ([#4699](https://github.com/micro/go-micro/issues/4699)) — Recent provider resilience work shipped retry cancellation and failure metadata, but the live AtlasCloud plan/delegate harness exposed a concrete malformed/nested tool-call recovery gap. Fixing it protects the agent loop's operability across providers while keeping the queue honest about current CI signals. Keep this to recovery/rejection behavior and regression coverage; surface any provider-contract or default-behavior changes for human review.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
