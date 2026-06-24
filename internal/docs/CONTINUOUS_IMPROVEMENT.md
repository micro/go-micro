# Continuous Improvement Loop

Go Micro is an agent harness. This file defines the **autonomous loop that builds
it** — the framework's own thesis (an agent operating a system) pointed at itself.
Claude Code drives the loop; Codex executes scoped tasks; the human sets direction
and can stop or revert anything at any time.

> **North Star.** Every increment must advance the thesis in [`THESIS.md`](THESIS.md):
> a holistic agent harness and service framework encapsulating the lifecycle of
> **services → agents → workflows**. Judge each change against it — work that
> doesn't move toward that lifecycle isn't an improvement, however clean.

## Autonomy

Full autonomy, **no approval gates**. Each increment: Claude Code picks the work,
implements it (or dispatches Codex), opens a PR, and **merges it** — including
reviewing and merging Codex's PRs. The only gate is **correctness**: `go build`,
`go test`, and `golangci-lint` must be green (that's not an approval, it's not
shipping broken code).

Transparency replaces approval: every increment ends with a one-line digest, and
every change is a small, reversible, single-concern PR the human can revert.

## What counts as an improvement

Grounded in real signal, never speculative rewrites. Each cycle draws from:

1. **Roadmap** — the Now/Next items in `ROADMAP.md` (harness depth: durable runs,
   observability, streaming, human-in-the-loop; hardening: resilience, conformance).
2. **Open issues** — the scoped backlog (e.g. #3010–#3014).
3. **Improvement radar** — a scan each cycle for: missing/weak tests, lint or
   quality issues, docs/code drift, and DX friction.
4. **Dogfooding** — actually build with the harness (`micro new` → `run` → `chat`,
   an agent + a flow) and fix what hurts. Friction found here is high-signal.

## The cycle (one increment)

1. Sync `master`.
2. If a Codex PR is open and CI-green → review (diff + gates + correctness vs its
   issue) and merge it.
3. Else pick the single highest-value item from the sources above.
4. Implement it, or dispatch to Codex (`@codex <instruction>` on the issue) if it's
   a well-scoped chunk and Codex is free. **Codex is serial — one task at a time.**
5. Verify `build`/`test`/`lint` locally.
6. Open a PR (one concern) and merge it.
7. Post a one-line digest; refresh the backlog from the radar.

## Roles

- **Claude Code** — orchestrator, implementer, reviewer, integrator, merger.
- **Codex** — serial builder for well-scoped chunks, dispatched via `@codex`.
- **Human** — sets direction; owns brand/positioning copy and breaking public-API
  decisions; can stop or revert anything.

## Guardrails

- One concern per PR; small and reversible.
- Stay on `claude/*` branches (Codex on `codex/*`); never two agents on one branch;
  base PRs on `master` (don't stack on an in-flight branch). See `CODEX.md`.
- **Off-limits without the human:** brand/positioning/marketing copy, breaking
  public API changes, product-default changes with broad behavioral impact, new
  dependencies, architectural rewrites. The loop proposes these in the digest; it
  does not merge them autonomously.

## Scheduling

- **In-session cron** (`CronCreate`) — runs increments while this Claude session is
  alive. Convenient, but the remote environment is reclaimed on inactivity and
  recurring jobs expire after 7 days, so it is **not** a durable scheduler.
- **GitHub Actions (durable)** — a scheduled workflow that runs the loop
  independently of any session. This is the real backbone; it opens a fresh
  tracking issue for each increment and dispatches Codex there, so every run gets
  a unique Codex-derived branch and a PR that closes its tracking issue. It needs
  a `CODEX_TRIGGER_TOKEN` repo secret from a user account Codex responds to;
  without that secret the workflow deliberately no-ops to avoid ignored bot
  comments. See `.github/workflows/continuous-improvement.yml`.

## Stop / redirect

- In-session: `CronDelete <id>` (or end the session).
- Durable: disable/delete the workflow.
- Or just tell Claude Code to pause or change focus — direction always wins over
  the loop.
