---
title: "Go Micro Joins OpenAI's Codex for Open Source"
linkTitle: "Joins OpenAI's Codex"
date: 2026-06-23
description: "OpenAI is backing Go Micro through Codex for Open Source. What the grant is, and what it means for a framework built around agentic development."
category:
- announcement
tags:
- AI
- sponsorship
- open-source
aliases:
  - /blog/29
---
Go Micro has been accepted into **OpenAI's Codex for Open Source** program. OpenAI is covering six months of ChatGPT Pro — and with it, access to Codex — to support the day-to-day work of maintaining the project.

## What the grant is

Codex for Open Source backs maintainers directly: the constant, unglamorous work of reviewing changes, cutting releases, triaging issues, and keeping quality high. That's most of what maintaining a framework actually is, and it's the part that rarely gets funded. OpenAI putting Codex against it is a real, practical help.

## Why it fits Go Micro

There's a neat symmetry here. Go Micro is a framework for **agentic development** — agents that use tools, services that are automatically AI-callable, workflows that [loop until the job is done](/docs/guides/agent-loops.html). Building and maintaining it with an agentic coding tool is that same idea pointed back at itself: tools calling tools.

It also lines up with what's already in the box. `openai` has been a first-class model provider in Go Micro since the AI-native rewrite — one of [several](/docs/guides/ai-provider-guide.html) behind the same `ai.Model` interface. So you can build *on* OpenAI models with Go Micro, and now we build Go Micro itself with OpenAI's tooling.

## What we'll use it for

Honestly: throughput. More reviewed PRs, faster releases, quicker issue triage, more examples and docs. The recent run of work — [agent loops](/docs/guides/agent-loops.html), [A2A](/blog/2026/06/18/agents-across-frameworks-a2a/), durable flows, a blocking lint gate in CI — is the pace we want to keep, and maintainer tooling is what sustains it.

## Thanks

Thanks to OpenAI for backing open source maintainers, and for supporting Go Micro specifically. It joins [Anthropic](/blog/2026/03/04/building-the-ai-native-future-of-go-micro-with-claude/) and [Atlas Cloud](/blog/2026/05/28/atlas-cloud-sponsors-go-micro-300-ai-models-one-integration/) — the companies helping keep this project moving.

