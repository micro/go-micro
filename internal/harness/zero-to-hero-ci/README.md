# 0→hero CI harness

This directory owns the no-secret reference scenario for the Go Micro
services → agents → workflows lifecycle. It is intentionally small and
scripted so CI can run it on every push without external services or model keys.

`run.sh` verifies three boundaries together:

1. **Run** — `micro run` remains available as the local development entry point.
2. **Chat** — `micro chat` remains available as the interactive agent entry point.
3. **Inspect** — `micro inspect agent <name>` and `micro inspect flow <name>`
   remain available as the local run-history inspection step, with `micro flow
   runs` preserving durable workflow history inspection.

After the CLI boundary smoke checks, the script runs the deterministic harnesses
that boot real services, agents, workflows, store-backed run history, and A2A
with only the LLM mocked.

## Local and CI entry points

The default GitHub harness workflow runs this script on every push and pull
request after the 0→1 scaffold contract. Developers can run the same no-secret
contract locally with:

```sh
make harness
```

That target intentionally exercises the documented getting-started path before
the 0→hero scenario, so the public scaffold → run/chat → inspect lifecycle stays
executable outside CI as well. Live provider checks remain separate and gated by
configured API keys (`make provider-conformance` or the scheduled/manual CI job).
