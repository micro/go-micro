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

1. **Fix plan-delegate duplicate-delegate replay test hang** ([#4527](https://github.com/micro/go-micro/issues/4527)) — This is the highest-value next increment because the autonomous loop's safety model depends on green, bounded CI. A mock duplicate-delegate regression currently can hang `richgo test -v -race -cover ./...` for the plan/delegate harness, so fixing service/agent/broker cleanup protects every later builder PR while staying tightly scoped to the services → agents lifecycle.
2. **Preserve A2A fallback artifact text for AtlasCloud** ([#4522](https://github.com/micro/go-micro/issues/4522)) — The completed A2A stream-fallback task can return empty text parts, which makes chat/inspect output and cross-agent handoff look successful while hiding the answer. This remains the top user-visible adoption gap because developer trust depends on a readable run → chat → inspect loop, especially when streaming falls back.
3. **Ensure AtlasCloud agent-flow sends onboarding notification** ([#4529](https://github.com/micro/go-micro/issues/4529)) — The event-driven onboarding flow can create the workspace but miss the required notification before timing out. This is a direct services → agents → workflows seam, so it belongs ahead of broader interop polish once CI is unblocked and fallback text is reliable.
4. **Make AtlasCloud universe A2A reachability probe deterministic** ([#4504](https://github.com/micro/go-micro/issues/4504)) — The universe checkout flow completed, but the A2A reachability probe timed out under AtlasCloud. This still matters for cross-framework operability and agent discoverability, but ranks after the active plan/delegate, fallback-content, and agent-flow regressions because those are more visible to the first-agent/0→hero lifecycle.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
