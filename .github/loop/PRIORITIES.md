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

1. **Trace agent RunInfo in OpenTelemetry spans** ([#4315](https://github.com/micro/go-micro/issues/4315)) — #4384 shipped the first-agent command/wayfinding harness and #4381 is closed, so the highest-value remaining gap is operability for people who make it past the on-ramp: connect existing run metadata to traces so real agent runs can be debugged across steps, tool calls, delegation, failures, services, and flows without inventing a new surface.
2. **Resume agent runs from checkpoints** ([#4368](https://github.com/micro/go-micro/issues/4368)) — Durable agent runs are the next lifecycle seam after observability: flows already checkpoint, and agents need a focused, non-breaking resume slice that preserves completed tool calls and avoids duplicate side effects before broader durability or API design work.
3. **Broaden provider streaming conformance** ([#4386](https://github.com/micro/go-micro/issues/4386)) — The blog says Anthropic streaming shipped, but the roadmap still calls for provider-backed streaming across chat and A2A. Add a focused, provider-gated conformance slice so streaming stays end-to-end rather than becoming a one-provider success story.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
