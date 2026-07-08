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

1. **CI-verify first-agent docs wayfinding links** ([#4291](https://github.com/micro/go-micro/issues/4291)) — Developer adoption is the current goal, and the README/website/examples now tell a coherent first-agent path: install troubleshooting → `micro agent demo`/`micro examples`/`micro zero-to-hero` → examples index → no-secret first agent → debugging → 0→hero. The highest-value next guard is preventing that on-ramp from drifting. Keep this scoped to a local-file check wired into an existing harness or Make target, with no new runtime dependencies.
2. **Fix AtlasCloud agent conformance guarded delegation** ([#4296](https://github.com/micro/go-micro/issues/4296)) — The plan-delegate notify recovery shipped in #4299, but the latest live provider conformance run exposed a separate AtlasCloud agent harness failure: missing guarded delegate behavior within the retry budget. This is the top remaining Now-phase reliability defect because guarded delegation is core to services → agents handoff safety and should be CI-verifiable without public API changes.
3. **Fix AtlasCloud A2A streaming fallback tool-call reporting** ([#4297](https://github.com/micro/go-micro/issues/4297)) — The same live provider run found the A2A streaming fallback path reporting neither tool-call evidence nor run metadata (`tool=false runInfo=false`). Rank it just behind guarded delegation: A2A streaming is an important interop seam, but the issue is narrower than the base agent conformance failure and should stay focused on surfacing tool/run evidence in the existing harness.
4. **Propagate cancellation and retry signals through provider model calls** ([#4273](https://github.com/micro/go-micro/issues/4273)) — After the adoption link guard and current live AtlasCloud failures are stable, the remaining Now-phase resilience seam is provider-boundary cancellation/deadline/retry behavior across the agent loop. Keep it focused on fakes/tests and avoid public API or architecture changes.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
