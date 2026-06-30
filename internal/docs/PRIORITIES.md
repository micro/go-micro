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

1. **Verification / grader loop for flows** ([#3435](https://github.com/micro/go-micro/issues/3435)) — the Now-phase failure/resilience and 0→hero contracts have largely shipped, and cross-provider agent scenario conformance closed via #3438. The highest-value remaining lifecycle gap is a CI-verifiable workflow primitive that can grade a step's output against a rubric, route failures back with feedback, and retry within bounded failure semantics. This keeps the services → agents → workflows story operable: known workflow paths get deterministic structure, while agent/model outputs get an explicit trust boundary instead of ad hoc prompt checks.

2. **Flow hill-climbing loop from run traces** ([#3439](https://github.com/micro/go-micro/issues/3439)) — after verification exists, use the run/trace foundation from recent telemetry and inspect work to analyze failures over time and propose prompt/grader improvements. Keep this behind the grader loop because hill-climbing depends on a stable evaluation signal; ranked second because it turns the harness's own continuous loop into a product-facing proof of durable, observable workflow improvement.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
