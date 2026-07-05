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

1. **Keep website quickstart on the first-agent on-ramp** ([#4071](https://github.com/micro/go-micro/issues/4071)) — the README, Getting Started guide, docs index, examples map, CLI docs, and zero-to-hero guide now tell the provider-free first-agent story in order; the website Quick Start page still trails that canonical path by omitting some wayfinding anchors (`micro agent demo`, the no-secret transcript, debugging/inspect). Update the page and extend the focused docs harness so quickstart drift fails fast. This keeps developer adoption weighted alongside hardening instead of letting the queue become purely internal.
2. **Add install troubleshooting to the first-agent on-ramp** ([#4077](https://github.com/micro/go-micro/issues/4077)) — after the quickstart page is aligned, close the next 0→1 adoption seam: install verification and recovery before a newcomer reaches `micro new` or `micro agent demo`. Link a focused install troubleshooting path from the canonical on-ramp pages, cover binary installer versus `go install`, PATH/version checks, and the no-secret smoke path, then add a docs harness assertion so the install-recovery link cannot drift out of the first-agent journey.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
