---
layout: default
---

# Provider Conformance Matrix

Go Micro treats model providers as interchangeable pieces of the same agent
harness: services expose tools, agents reason over them, and workflows stitch the
work together. The conformance harness keeps that promise honest by running the
same deterministic services → agents → workflows scenarios against every
configured provider.

The live harness is in `internal/harness/provider-conformance`. It skips
providers without API keys by default, so it is safe to run locally, and it fails
when any configured provider breaks the shared contract.

```sh
go run ./internal/harness/provider-conformance
```

For a no-key smoke test of the same harness wiring, run the mock provider:

```sh
go run ./internal/harness/provider-conformance -providers mock
```

## Status legend

| Status | Meaning |
| --- | --- |
| ✅ Verified | Covered by the provider-conformance harness for configured live providers. |
| ⚠️ Unverified | Implemented in the public API, but not yet exercised by provider conformance. |
| — Unsupported | Not exposed by that provider integration today. |

## Harness coverage by capability

These rows describe what the conformance harness verifies today. A provider is
considered conformant when the configured-key run passes all selected harnesses.

| Capability | Harness coverage | Notes |
| --- | --- | --- |
| Simple generation | ✅ Verified | Each harness asks the provider to produce an agent response through `ai.Model`. |
| Service tool calls | ✅ Verified | Harness services are discovered and invoked as model-selected tools. |
| Multi-step tool use | ✅ Verified | The `universe` and `plan-delegate` harnesses require more than one service/tool action. |
| `plan` | ✅ Verified | `plan-delegate` verifies that the conductor agent stores a plan in scoped state. |
| `delegate` | ✅ Verified | `plan-delegate` verifies agent-to-agent delegation over real RPC. |
| Guardrail/stop behavior | ✅ Verified | `universe` runs with guardrails enabled and asserts the guarded path completes. |
| Streaming | ⚠️ Unverified | `ai.Model.Stream` exists on the interface, but end-to-end streaming conformance is a roadmap item. |
| Structured errors | ⚠️ Unverified | Error handling is covered by normal test suites, but provider conformance does not yet compare structured provider errors. |

## Provider capability matrix

This matrix combines the registered provider interfaces with the conformance
coverage above. The chat/text column is the harness path: when the provider has a
configured key, the conformance command exercises the verified rows in the
previous section.

| Provider | Chat/text agent harness | Image | Video | Streaming | Structured errors |
| --- | --- | --- | --- | --- | --- |
| `anthropic` | ✅ Verified when configured | — Unsupported | — Unsupported | ✅ Verified when configured | ⚠️ Unverified |
| `openai` | ✅ Verified when configured | ✅ Registered | — Unsupported | ⚠️ Unverified | ⚠️ Unverified |
| `gemini` | ✅ Verified when configured | — Unsupported | — Unsupported | ⚠️ Unverified | ⚠️ Unverified |
| `groq` | ✅ Verified when configured | — Unsupported | — Unsupported | ⚠️ Unverified | ⚠️ Unverified |
| `mistral` | ✅ Verified when configured | — Unsupported | — Unsupported | ⚠️ Unverified | ⚠️ Unverified |
| `together` | ✅ Verified when configured | — Unsupported | — Unsupported | ⚠️ Unverified | ⚠️ Unverified |
| `atlascloud` | ✅ Verified when configured | ✅ Registered | ✅ Registered | ⚠️ Unverified | ⚠️ Unverified |

## Running a focused check

Use `-providers` to select a provider and `-harnesses` to narrow the scenario:

```sh
go run ./internal/harness/provider-conformance \
  -providers openai,anthropic \
  -harnesses agent-flow,plan-delegate
```

By default missing live-provider keys are reported as skips. Add
`-require-configured` in CI when a selected provider must be present:

```sh
go run ./internal/harness/provider-conformance \
  -providers openai \
  -require-configured
```

The command also prints the registered model, image, and video provider
capabilities before running conformance. Disable that with `-capabilities=false`
when you only want pass/fail output.

For automation, add `-summary-json` to capture the selected providers,
harnesses, registered capability rows, and pass/skip/fail results in a stable
machine-readable file. Add `-capabilities-markdown` when you also want a
ready-to-publish Markdown support table for release notes, docs, or issue
updates:

```sh
go run ./internal/harness/provider-conformance \
  -providers mock \
  -summary-json provider-conformance-summary.json \
  -capabilities-markdown provider-capabilities.md
```

## Related docs

- [The Agent Harness](agent-harness.html)
- [Agents and Workflows](agents-and-workflows.html)
- [AI Provider Guide](ai-provider-guide.html)
- [Roadmap](/docs/roadmap.html)
