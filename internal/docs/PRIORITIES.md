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

1. **Execution lifecycle hooks & metadata** (#2980) — before/after-tool, retry,
   and failure hooks; first reconcile the issue with the shipped `AgentWrapTool`,
   structured refusal reasons, `RunInfo`, OpenTelemetry spans, durable resume, and
   A2A streaming task lifecycle, then close it or scope only a CI-verifiable gap
   that those primitives cannot express. Roadmap → resilience/operability; now
   ranked first because streaming shipped and this is the remaining open Now-phase
   reliability seam for long-running agents and workflows.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
