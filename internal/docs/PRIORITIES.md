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

1. **CI-verify cross-provider agent scenario conformance** ([#3432](https://github.com/micro/go-micro/issues/3432)) — failure-summary classification shipped via #3430 and closed the cancellation/retry contract (#3425), after provider-backed `ai.Stream` conformance (#3423), agent OpenTelemetry (#3418), durable checkpoint resume (#3414), A2A streaming fallback (#3410), the consolidated developer-flow harness (#3399), and the deploy checkpoint in the 0→hero harness (#3384). With the main failure/resilience and 0→hero contracts now covered, the highest-value remaining Now-phase gap is proving the same services → agents → workflows scenario across providers: tool discovery/calls, memory/run metadata, streaming fallback, and flow/gateway boundaries need one no-secret CI contract plus key-gated live-provider checks so builders do not experience provider-specific seams.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
