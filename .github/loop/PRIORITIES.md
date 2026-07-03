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

1. **Make first-agent preflight failures actionable** ([#3841](https://github.com/micro/go-micro/issues/3841)) — developer adoption remains the current goal, and the README/docs now provide a no-secret first-agent route, `micro agent preflight`, a debugging guide, docs wayfinding, and a 0→hero harness. With #3544 closed by #3845, the highest-value remaining Now-phase item is making the first local failure feel like a guided doctor: when Go, the CLI, provider-key setup, or the gateway port is wrong, the CLI should say exactly what to fix and where to continue, without needing secrets or provider calls.
2. **Emit OpenTelemetry spans for agent run timelines** ([#3525](https://github.com/micro/go-micro/issues/3525)) — runs are increasingly inspectable and recent work correlated trace metadata, verified restart boundaries, added retry/cancellation hardening, and narrowed provider failures to diagnosability gaps. The next highest-value Next-phase internal increment is to turn the `RunInfo` foundation into standard OTel spans for agent runs, model calls, tool calls, checkpoint/resume, cancellation/deadlines, and failures so operators can understand the service → agent → workflow lifecycle in production.
3. **Resume long-running agent runs from checkpoints** ([#3847](https://github.com/micro/go-micro/issues/3847)) — the roadmap and blog promise a harness that recovers from failure, and flows already checkpoint/resume. After the adoption preflight and observability foundation, durable agent resume closes the most visible lifecycle seam between flows and agents: long agent work should survive restarts without replaying completed tool side effects or resuming canceled/expired runs.
4. **Add an AP2 mandate layer over A2A and x402** ([#3552](https://github.com/micro/go-micro/issues/3552)) — this is a forward interop investment, not a Now-phase blocker: Go Micro already has A2A agents and x402 paid tools, so a small signed-mandate foundation can keep agent payments aligned with the open-protocol story without pulling the queue away from adoption, resilience, observability, or durable agent operation. Keep it additive and opt-in while the AP2/FIDO work settles.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
