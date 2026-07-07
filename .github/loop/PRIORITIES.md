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

1. **Fix AtlasCloud plan-delegate notify handoff recovery** ([#4277](https://github.com/micro/go-micro/issues/4277)) — Recent AtlasCloud tool-call parsing, duplicate-replay, first-agent harness, and universe broker fixes landed (#4257/#4262/#4270/#4279/#4289), but live provider conformance still exposed a narrow plan/delegate recovery gap where the delegated notify side effect is missing. This is now the highest-value Now-phase reliability defect: the mock 0→hero harness is stable again, and live provider parity is the remaining confidence gap for services → agents → workflows.
2. **CI-verify first-agent docs wayfinding links** ([#4291](https://github.com/micro/go-micro/issues/4291)) — Developer adoption is the current goal, and recent README/website/examples work added a strong first-agent path; the next highest-value adoption guard is preventing link drift across install troubleshooting, examples, no-secret first-agent, debugging, and 0→hero docs. Keep this scoped to a local-file check wired into an existing harness or Make target, with no new runtime dependencies.
3. **Propagate cancellation and retry signals through provider model calls** ([#4273](https://github.com/micro/go-micro/issues/4273)) — After the live AtlasCloud handoff gap and the first-agent wayfinding guard are stable, the remaining Now-phase resilience seam is provider-boundary cancellation/deadline/retry behavior across the agent loop. Keep it focused on fakes/tests and avoid public API or architecture changes.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
