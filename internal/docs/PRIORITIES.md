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

1. **Add memory compaction and retrieval for long-running agents** ([#3321](https://github.com/micro/go-micro/issues/3321)) — with durable resume, broad streaming, CLI run inspection, and RunInfo trace correlation now shipped, tackle the leading Later-roadmap gap: bounded, store-backed memory that keeps long agent runs useful without turning Go Micro into a prompt-layer framework.
2. **Add human-in-the-loop pause and resume for agent workflows** ([#3329](https://github.com/micro/go-micro/issues/3329)) — after memory is bounded, close the next operability gap for scheduled/looping work: durable pending-input states that let humans approve or provide context and then resume the same service/agent/workflow runtime.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
