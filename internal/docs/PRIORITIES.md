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

1. **Add durable checkpoint/resume for agent runs** ([#3524](https://github.com/micro/go-micro/issues/3524)) — the 0→hero reference app has now shipped, and the next highest-value lifecycle gap is making long-running agent work survive restarts the way flows already do. This is the clearest bridge from services → agents → workflows: flows can already checkpoint deterministic orchestration, but the dynamic agent loop still needs a CI-verifiable resume contract so scheduled, looping agents can be operated rather than merely invoked.
2. **Emit OpenTelemetry spans for agent run timelines** ([#3525](https://github.com/micro/go-micro/issues/3525)) — recent work made runs inspectable and correlated trace metadata through scheduled dispatch; the next step is to turn that RunInfo foundation into standard OTel spans for agent runs, model calls, and tool calls. This keeps `micro runs` useful while making the harness observable in the systems developers already run.
3. **Add retry and timeout resilience to agent tool execution** ([#3526](https://github.com/micro/go-micro/issues/3526)) — flow retry/backoff and cancellation safety have shipped, but the agent loop still needs the same failure semantics around tool/model calls: bounded retries, deadline propagation, cancellation, and visible retry/timeout outcomes. This belongs high because operability is the difference between an agent demo and a dependable service.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
