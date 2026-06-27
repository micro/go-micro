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

## Now (ranked)

1. **Cross-provider agent conformance matrix** (#3187) — run the same agent
   scenario across supported providers behind credential gates, with deterministic
   local skips and a fake/local provider path. Roadmap → *Now*; highest leverage
   trust work after streaming shipped because every agent feature depends on
   provider behavior staying consistent.
2. **Registry disconnection detection** (#2956) — readiness/health when a service
   silently loses its registry connection. Roadmap → *Now* failure/resilience;
   keeps the service substrate operable because agents depend on discovery.
3. **Agent observability spans** (#3182) — export `RunInfo` as OpenTelemetry spans
   for agent runs, model calls, tool calls, delegation, and failures. Roadmap →
   *Next*; makes the now-durable and streaming harness inspectable in production.
4. **Execution lifecycle hooks & metadata** (#2980) — before/after-tool, retry,
   and failure hooks; first check overlap with the shipped run-timeline /
   OpenTelemetry work and scope to what's not already covered.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
