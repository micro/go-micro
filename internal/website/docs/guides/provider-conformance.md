---
layout: default
---

# Provider Conformance Matrix

Go Micro keeps providers behind one `ai.Model` interface, but production agents
need to know which behaviors are actually covered by the harness checks. This
matrix records the provider capabilities that are either verified by
`internal/harness/provider-conformance` or known to be outside the current
provider surface.

## How to read the matrix

| Mark | Meaning |
|---|---|
| ✅ Verified | Covered by the conformance harness when the provider API key is configured. |
| ⚠️ Unverified | Implemented in the shared harness path, but not yet checked by a dedicated provider conformance case. |
| — Unsupported | Not implemented by the provider or by the current shared `ai.Model` interface. |

The live harness runs the same deterministic examples against every configured
provider:

```sh
go run ./internal/harness/provider-conformance
```

For local development without provider keys, run the mock path:

```sh
go run ./internal/harness/provider-conformance -providers mock
```

## Matrix

| Capability | Anthropic | OpenAI | Gemini | Groq | Mistral | Together | Atlas Cloud |
|---|---|---|---|---|---|---|---|
| Simple generation | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified |
| Service tool calls | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified |
| Multi-step tool use | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified |
| `plan` | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified |
| `delegate` | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified | ✅ Verified |
| Guardrail / stop behavior | ⚠️ Unverified | ⚠️ Unverified | ⚠️ Unverified | ⚠️ Unverified | ⚠️ Unverified | ⚠️ Unverified | ⚠️ Unverified |
| Streaming | — Unsupported | — Unsupported | — Unsupported | — Unsupported | — Unsupported | — Unsupported | — Unsupported |
| Structured errors | ⚠️ Unverified | ⚠️ Unverified | ⚠️ Unverified | ⚠️ Unverified | ⚠️ Unverified | ⚠️ Unverified | ⚠️ Unverified |

## What the current harness verifies

The default conformance run executes these harnesses for every selected provider
with an API key:

- `universe` — service discovery and service-backed tool calls.
- `agent-flow` — a workflow dispatching work through an agent.
- `plan-delegate` — the built-in `plan` and `delegate` tools on the agent loop.

That means simple generation, service tool calls, multi-step tool use, planning,
and delegation are verified together as one services → agents → workflows path.
Provider keys are optional by default so scheduled or local runs can skip missing
providers without turning the entire check red; use `-require-configured` when a
CI job should fail for missing secrets.

## Keeping this page current

When a provider or harness changes, update this page in the same PR as the
conformance change. In particular:

- Move a capability from ⚠️ to ✅ only when a provider conformance harness asserts
  the behaviour for that provider.
- Move a capability from — to ✅ or ⚠️ only when the `ai` package exposes the
  relevant interface and at least one provider implements it.
- Keep provider additions in sync with `ai.CapabilityRows()` and the registered
  providers imported by `internal/harness/provider-conformance`.

## Related docs

- [The Agent Harness](agent-harness.html)
- [Agents and Workflows](agents-and-workflows.html)
- [AI Provider Guide](ai-provider-guide.html)
- [Roadmap](/docs/roadmap.html)
