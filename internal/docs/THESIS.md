# Go Micro — Thesis & North Star

This is the North Star for the project and for the autonomous improvement loop
(see `CONTINUOUS_IMPROVEMENT.md`). Every change should move toward it; work that
doesn't isn't an improvement, however clean.

## Thesis

Go Micro is an **agent harness and service framework** — one runtime that, holistically,
encapsulates the **lifecycle of services, agents, and workflows**. Not three
products stitched together: one set of primitives, because an agent is a
distributed system and building one is building a service.

## The progression: services → agents → workflows

Value is unlocked in order, and each layer needs the one beneath it:

1. **Services** — typed, discoverable, callable capabilities. The substrate; every
   endpoint is automatically an AI-callable tool.
2. **Agents** — a model with memory and tools that *uses* those services, plans,
   delegates, and is bounded by guardrails. Intelligence on top of capability.
3. **Workflows** — the part that **pieces it all together**: composing agents and
   services over time, deterministically where the path is known and dynamically
   where it isn't, on schedules and in loops. The workloads come *after* the
   agents, because the value is in stitching it into systems that do real work.

A harness that stops at "a model in a loop" is incomplete. The point is the whole
lifecycle — capability, intelligence, and orchestration as one runtime.

## Why now

The frontier is moving from chat to **scheduled, looping, work-performing agents**:
Anthropic itself is building toward agents that do work on a cadence (Claude for
Work, schedulers), and running coding agents *continuously in loops* is becoming
standard practice among the people who build them. That shift is exactly the
"workflows after agents" layer — and the harness is what makes it safe, durable,
observable, and composable instead of a fragile script.

The bet: whoever gives Go a holistic harness for the **whole lifecycle** — not just
an agent SDK, not just a service framework — owns where agentic software gets built.

## What every improvement should serve

Judge each loop increment against the North Star:

1. **Make the harness real** — operate the loop in production: durability,
   observability, resilience, streaming, human-in-the-loop.
2. **Tighten the lifecycle** — services ↔ agents ↔ workflows as one runtime, not
   three silos.
3. **Advance orchestration** — durable, resumable, scheduled, looping workflows
   that compose agents and services over time.
4. **Sharpen DX** — the 0→1 and 0→hero paths stay effortless.
5. **Strengthen interop** — MCP (tools), A2A (agents), x402 (paid tools).
6. **Harden trust** — cross-provider conformance, failure semantics, tests.

Prefer changes that advance these; avoid scope that doesn't. Brand/positioning
copy and breaking public-API changes stay with the human.

## The loop is the proof

Go Micro is built by an autonomous agentic loop — Claude Code and Codex
continuously improving the repo against this North Star. That isn't a gimmick; it's
the thesis applied to itself: an agent harness, built by agents running in a loop.
If the harness is good enough to build itself, it's good enough to build your
agentic software.

## What this is not

The framework is the product — no hosted platform, no enterprise tier, no VC, no
graph DSL. Sustained by sponsorship from those who run it. See `ROADMAP.md`.
