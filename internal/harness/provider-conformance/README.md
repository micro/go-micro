# Provider conformance

This harness keeps the services → agents → workflows lifecycle honest across the
supported AI providers. It runs the same end-to-end scenarios against each
configured provider and treats missing provider keys as an explicit skip, so the
suite is safe for local development, forks, and scheduled CI.

## What it exercises

`go run ./internal/harness/provider-conformance` fans out over the provider-facing
agent test and the harnesses in `internal/harness`:

- `agent` — provider tool-call conformance through `agent.Ask`, including run metadata propagation.
- `universe` — service discovery plus agent tool calls over the real runtime.
- `agent-flow` — a workflow event that drives an agent to call services.
- `plan-delegate` — plan persistence plus agent-to-agent delegation and service
  calls.
- `a2a-stream-fallback` — A2A `message/stream` through the gateway, including
  fallback from unsupported provider streaming to the tool-calling `Ask` path while
  preserving run metadata.

The command also emits the registered provider capability matrix so the run shows
which providers advertise model, image, video, and streaming support.

## Local usage

Run the deterministic path with no secrets:

```sh
go run ./internal/harness/provider-conformance -providers mock
```

Run every live provider that has a key in the environment:

```sh
go run ./internal/harness/provider-conformance \
  -summary-json provider-conformance-summary.json \
  -summary-markdown provider-conformance-summary.md \
  -capabilities-markdown provider-capabilities.md
```

Provider keys are read from `MICRO_AI_API_KEY` or the provider-specific variable:

| Provider | Secret / environment variable |
| --- | --- |
| Anthropic | `ANTHROPIC_API_KEY` |
| OpenAI | `OPENAI_API_KEY` |
| Gemini | `GEMINI_API_KEY` |
| Groq | `GROQ_API_KEY` |
| Mistral | `MISTRAL_API_KEY` |
| Together | `TOGETHER_API_KEY` |
| AtlasCloud | `ATLASCLOUD_API_KEY` |

Use `-require-configured` when you want a selected provider without a key to fail
instead of skip:

```sh
go run ./internal/harness/provider-conformance \
  -providers anthropic,openai \
  -require-configured
```

## Scheduled CI behavior

The `Harness (E2E)` workflow runs on pushes and pull requests with deterministic
mock LLMs, including `provider-conformance -providers mock`. On the daily schedule and manual dispatch it also runs the live
provider conformance job. That job:

1. runs the same `agent`, `universe`, `agent-flow`, `plan-delegate`, and `a2a-stream-fallback` harness list,
2. reads the provider keys from repository secrets,
3. skips providers whose secrets are absent,
4. fails when any configured provider fails a harness, and
5. uploads JSON and Markdown coverage artifacts for the run.

The job also appends the Markdown summary and capability matrix to the GitHub
Actions step summary, making configured, skipped, and failed provider coverage
visible without downloading artifacts.

## Adding a provider

To bring a new provider into scheduled conformance:

1. register its `ai` provider implementation and capability metadata,
2. add the provider name and key variable to `providerEnv` in `main.go`,
3. import the provider package in `main.go`,
4. pass the matching repository secret through `.github/workflows/harness.yml`,
   and
5. run `go run ./internal/harness/provider-conformance -providers <name> \
   -require-configured` with a live key before opening the change.
