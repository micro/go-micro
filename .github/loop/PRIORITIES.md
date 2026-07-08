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

1. **Fix AtlasCloud Minimax tool follow-up retry after tool results** ([#4355](https://github.com/micro/go-micro/issues/4355)) — The native service-tool 400 fallback (#4354/#4361) has shipped, so the remaining Now-phase live-provider break is the follow-up retry transcript after a tool result. This protects plan/delegate plus service side effects from failing after partial work, directly supporting the services → agents → workflows lifecycle under real provider behavior.
2. **CI-verify the Your First Agent chat walkthrough** ([#4363](https://github.com/micro/go-micro/issues/4363)) — Developer adoption is the current goal, and the README/website now promise a walkable first-agent path. A no-secret CI check for the documented `micro chat` and inspect/history boundary keeps the 0→1 agent on-ramp from drifting while remaining small, reversible, and public-API neutral.
3. **Trace agent RunInfo in OpenTelemetry spans** ([#4315](https://github.com/micro/go-micro/issues/4315)) — After the live conformance regression and first-agent walkthrough contract are stable, the highest Next-phase operability gap is connecting existing run metadata to traces so real agent runs can be debugged across steps, tool calls, delegation, failures, services, and flows without inventing a new surface.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
