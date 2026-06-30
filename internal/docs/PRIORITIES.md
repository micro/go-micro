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

1. **Verify agent cancellation and retry semantics end to end** ([#3425](https://github.com/micro/go-micro/issues/3425)) — provider-backed `ai.Stream` conformance shipped via #3423 after agent OpenTelemetry (#3418), durable checkpoint resume (#3414), A2A streaming fallback (#3410), the consolidated developer-flow harness (#3399), and the deploy checkpoint in the 0→hero harness (#3384). The highest-value remaining Now-phase gap is failure and resilience: context deadlines, cancellation, retry/backoff, rate-limit/provider transient errors, and retry exhaustion need one CI-verifiable contract across model calls, agent tool execution/delegation, gateway paths, and flow boundaries so unattended service → agent → workflow runs fail safely and visibly.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
