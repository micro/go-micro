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

1. **Stabilize plan-delegate after unfinished delegation** ([#4322](https://github.com/micro/go-micro/issues/4322)) — This is the remaining Now-phase conformance failure in the multi-agent 0→hero path: real services and durable flow state can advance, but the delegated notification step may remain unfinished after partial side effects. Stabilizing that recovery keeps plan/delegate from being a brittle demo seam and protects the service → agent → workflow lifecycle under live-provider variance.
2. **CI-verify first-agent CLI wayfinding commands** ([#4336](https://github.com/micro/go-micro/issues/4336)) — Developer adoption is the current goal, and the README/docs now steer new users through installed CLI affordances (`micro agent demo`, `micro examples`, `micro zero-to-hero`) before deeper tutorials. Those commands should be treated as a no-secret on-ramp contract so the binary keeps pointing developers to the maintained first-agent, debugging, examples, and 0→hero path instead of drifting silently.
3. **Trace agent RunInfo in OpenTelemetry spans** ([#4315](https://github.com/micro/go-micro/issues/4315)) — Once the live harness failure above recovers reliably and the CLI on-ramp is protected, the highest Next-phase operability gap is connecting existing run metadata to traces so real agent runs can be debugged across steps, tool calls, delegation, failures, services, and flows without inventing a new surface.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
