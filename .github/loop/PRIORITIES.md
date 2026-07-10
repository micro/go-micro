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

1. **CI-verify first-agent docs wayfinding stays in sync** ([#4561](https://github.com/micro/go-micro/issues/4561)) — The README, website index/quickstart, examples map, and no-secret first-agent/0→hero docs now tell a stronger adoption story, but the loop needs a small deterministic guard so those surfaces do not drift apart. This is the highest-value open gap because the current goal is developer adoption: the scaffold → run → first agent → debug → 0→hero path must stay walkable and discoverable, not just documented once.
2. **Ensure AtlasCloud agent-flow sends onboarding notification** ([#4529](https://github.com/micro/go-micro/issues/4529)) — The event-driven onboarding flow can create the workspace but miss the required notification before timing out. This remains a direct services → agents → workflows seam and a developer-trust issue for 0→hero because the workflow appears partly successful while a required side effect is absent.
3. **Make AtlasCloud universe A2A reachability probe deterministic** ([#4504](https://github.com/micro/go-micro/issues/4504)) — The universe checkout flow completes, but the A2A reachability probe can time out under AtlasCloud. This still matters for cross-framework operability and agent discoverability, but ranks after side-effect execution gaps because it is currently isolated to post-flow reachability rather than lost business effects.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
