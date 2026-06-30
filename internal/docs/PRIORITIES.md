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

1. **Add cross-provider agent conformance harness** ([#3491](https://github.com/micro/go-micro/issues/3491)) — the verification loop shipped in #3489, so the next highest-value Now-phase gap is confidence that the same services → agents → workflows contract works across providers. The README and website sell one harness with pluggable providers, while the roadmap calls out that each provider has its own tool-call loop; a shared no-secret/live-key conformance scenario is the fastest way to prevent provider drift from undermining the 0→hero path.

2. **Harden agent loop failure and cancellation semantics** ([#3492](https://github.com/micro/go-micro/issues/3492)) — once provider behavior is comparable, the remaining Now-phase operational risk is whether real runs fail safely: timeouts, rate limits, cancellation, deadline propagation, retry/backoff, and inspectable terminal state. This keeps the harness dependable under production conditions without adding a new product surface.

3. **Add a scheduled agent run harness contract** ([#3486](https://github.com/micro/go-micro/issues/3486)) — scheduled, looping, work-performing agents remain the right Next-phase cohesion target after hardening. This should compose existing services, agents, flows, store-backed memory, verification, and run inspection so the scaffold → run → chat → inspect lifecycle extends to unattended work without adding a hosted scheduler or breaking public APIs.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
