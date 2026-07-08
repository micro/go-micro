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

1. **Propagate cancellation and retry signals through provider model calls** ([#4273](https://github.com/micro/go-micro/issues/4273)) — The live AtlasCloud tool-call reporting defect closed with #4313, leaving provider-boundary resilience as the highest-value Now-phase hardening seam. Keep the increment focused on cancellation/deadline/retry behavior across model calls, with fakes/tests and no public API or architecture changes.
2. **CI-verify zero-to-hero deploy dry-run path** ([#4309](https://github.com/micro/go-micro/issues/4309)) — Developer adoption remains the current goal, and the docs now have a strong no-secret first-agent path; the next adoption gap is making sure the documented scaffold → run → chat → inspect → deploy dry-run lifecycle is continuously verified through the deploy boundary, not just described. This keeps the 0→hero contract honest alongside internal resilience work.
3. **Trace agent RunInfo in OpenTelemetry spans** ([#4315](https://github.com/micro/go-micro/issues/4315)) — After the Now-phase resilience and getting-started contract gaps, the highest Next-phase operability gap is connecting existing run metadata to traces so real agent runs can be debugged across steps, tool calls, delegation, failures, services, and flows without inventing a new surface.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
