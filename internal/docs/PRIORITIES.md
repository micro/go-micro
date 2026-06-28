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

## Developer experience (ranked)

1. **Harden agent failure and cancellation semantics** ([#3296](https://github.com/micro/go-micro/issues/3296)) — cross-provider conformance scheduling has shipped, leaving failure/resilience as the highest-value Now-roadmap gap. The harness now has stronger streaming and run-inspection foundations, so the next coherence risk is inconsistent timeout, cancellation, rate-limit, and retry/backoff behavior across agent, AI provider, service-tool, and flow boundaries.
2. **Expose run inspection in the CLI inner loop** ([#3297](https://github.com/micro/go-micro/issues/3297)) — the canon promises scaffold → run → chat → inspect → deploy, and recent work added run timelines, trace correlation, and a `micro runs` foothold; this remains the most visible DX gap until local agent/flow activity is documented and CI-tested as an actionable inspect step.
3. **Add durable agent run checkpoint and resume** ([#3306](https://github.com/micro/go-micro/issues/3306)) — once the remaining Now hardening/inspection seams are closed, the highest-value Next-roadmap item is making agent loops resumable like flows so long-running work can survive restarts without unsafe replay or hidden state loss.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
