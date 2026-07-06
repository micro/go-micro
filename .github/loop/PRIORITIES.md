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

1. **Make AtlasCloud delegated plan notifications exact-once** ([#4163](https://github.com/micro/go-micro/issues/4163)) — #4179 shipped the highest-value adoption gap by adding durable `micro loop` README and website wayfinding, so the top remaining Now-phase gap is the recurring live provider harness failure: AtlasCloud `plan-delegate` completes the work but emits duplicate delegated notifications before the exact-once guard. Fixing this keeps the services → agents → workflows lifecycle dependable under real provider behavior and unblocks the scheduled evaluator without broad API changes.
2. **Propagate cancellation and retry signals through provider model calls** ([#4175](https://github.com/micro/go-micro/issues/4175)) — after the exact-once delegated side-effect seam is closed, the next Now-phase reliability gap is failure handling under real provider conditions: cancellation/deadline propagation and retry/backoff must not duplicate tool side effects. This keeps the same services → agents → workflows lifecycle dependable across providers without changing public APIs.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
