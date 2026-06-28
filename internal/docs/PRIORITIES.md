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

1. **Schedule cross-provider agent conformance** ([#3295](https://github.com/micro/go-micro/issues/3295)) — after the 0→hero reference and provider-focused unit coverage shipped, the highest-value remaining Now-roadmap hardening gap is a scheduled, key-gated provider matrix that proves the same agent/tool workflow keeps working across supported models without blocking contributors who lack secrets.
2. **Harden agent failure and cancellation semantics** ([#3296](https://github.com/micro/go-micro/issues/3296)) — the harness is increasingly durable, streaming, observable, and human-in-the-loop; the next operability seam is making timeouts, cancellation, rate-limit errors, and retry/backoff behavior predictable across agent, AI provider, service-tool, and flow boundaries.
3. **Expose run inspection in the CLI inner loop** ([#3297](https://github.com/micro/go-micro/issues/3297)) — the canon promises scaffold → run → chat → inspect → deploy, and recent work has strengthened scaffold/run/chat/deploy; inspection remains the most visible DX gap for turning agent/flow activity into actionable breadcrumbs during local development.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
