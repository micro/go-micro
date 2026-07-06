# Agent provider conformance matrix

`go test ./...` includes `TestAgentProviderConformanceMatrix`, a shared agent
scenario that runs against every registered chat provider. The scenario asks an
agent to call a deterministic local tool, verifies the tool receives `ai.RunInfo`,
and checks the final response carries the conformance marker. A fake provider path
runs on every machine without network access so CI always exercises the harness.

Live providers are opt-in to avoid flaky unauthenticated PR checks and accidental
API spend. To run the live matrix, set `GO_MICRO_AGENT_CONFORMANCE_LIVE=1` plus the
provider API keys you want to exercise:

| Provider | Required API key | Optional model override |
| --- | --- | --- |
| OpenAI | `OPENAI_API_KEY` | `GO_MICRO_CONFORMANCE_OPENAI_MODEL` |
| Anthropic | `ANTHROPIC_API_KEY` | `GO_MICRO_CONFORMANCE_ANTHROPIC_MODEL` |
| Atlas Cloud | `ATLASCLOUD_API_KEY` | `GO_MICRO_CONFORMANCE_ATLASCLOUD_MODEL` |
| Gemini | `GEMINI_API_KEY` | `GO_MICRO_CONFORMANCE_GEMINI_MODEL` |
| Groq | `GROQ_API_KEY` | `GO_MICRO_CONFORMANCE_GROQ_MODEL` |
| Mistral | `MISTRAL_API_KEY` | `GO_MICRO_CONFORMANCE_MISTRAL_MODEL` |
| Together | `TOGETHER_API_KEY` | `GO_MICRO_CONFORMANCE_TOGETHER_MODEL` |

When `GO_MICRO_AGENT_CONFORMANCE_LIVE` or a provider key is absent, the live
provider subtest reports a deterministic skip. When both are present, a provider
failure is a real test failure because drift in chat, tool calling, run metadata,
or final-answer behavior means the services → agents lifecycle is no longer
consistent across providers.

The companion `TestAgentProviderConformanceFakeError` keeps provider error
propagation covered locally without relying on external credentials.

## Local no-secret conformance

Use `make provider-conformance-mock` to run the same provider conformance harness
through the deterministic mock provider. That target requires no API keys and is
what `make harness` delegates to after the 0→1 and 0→hero scenarios, so every PR
continues to exercise the provider-facing agent/tool contract without spending
live model credits.

Use `make provider-conformance` when you want the live-provider sweep: providers
without keys are skipped, and configured providers must satisfy the same harness
contract.

## Scheduled CI

The hourly/manual `Harness (E2E)` workflow runs the same matrix with
`GO_MICRO_AGENT_CONFORMANCE_LIVE=1` and the provider secrets exported. Providers
whose keys are absent still skip cleanly, while any configured provider must pass
the shared tool-calling scenario. This keeps scheduled conformance key-gated: PR
checks stay deterministic and no-key environments remain green, but maintained
provider credentials exercise the live matrix regularly.
