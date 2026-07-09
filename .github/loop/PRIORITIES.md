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

1. **Add first-agent docs command parity check** ([#4495](https://github.com/micro/go-micro/issues/4495)) — The current adoption goal says the queue must not collapse into internal hardening only. README, website docs, and recent blog posts now tell a coherent no-secret first-agent story (`micro agent demo` → examples → zero-to-hero), but that story can drift from CLI output and maintained example paths. Add a provider-free, CI-verifiable parity check so the 0→1/0→hero on-ramp remains walkable as the CLI and docs evolve.
2. **Add agent model retry/backoff contract** ([#4500](https://github.com/micro/go-micro/issues/4500)) — With the AtlasCloud missing-marker conformance gap closed by #4498, the highest-value remaining Now-phase hardening gap is provider-free coverage for transient model failures, retry budgets, backoff, and cancellation in the agent loop. This keeps cross-provider reliability moving without pushing another live-provider-only task ahead of the adoption on-ramp.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
