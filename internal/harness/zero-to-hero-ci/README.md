# 0→hero CI harness

This directory owns the no-secret reference scenario for the Go Micro
services → agents → workflows lifecycle. It is intentionally small and
scripted so CI can run it on every push without external services or model keys.

`run.sh` verifies five boundaries together:

1. **First agent** — `micro new`, `micro agent preflight`, `micro run`,
   `micro chat`, and `micro inspect agent <name>` remain available as the
   documented first-agent walkthrough path.
2. **Run** — `micro run` remains available as the local development entry point.
3. **Chat** — `micro chat` remains available as the interactive agent entry point.
4. **Inspect** — `micro inspect agent <name>` and `micro inspect flow <name>`
   remain available as the local run-history inspection step, with `micro flow
   runs` preserving durable workflow history inspection.
5. **Deploy** — `micro deploy --dry-run <target>` remains available as the
   deployment-boundary checkpoint. The dry run resolves configured deploy targets
   and services and prints the remote build/copy/systemd/health plan without
   building binaries, opening SSH connections, running `rsync`, or touching
   remote infrastructure.

After the CLI boundary smoke checks, the script runs the deterministic harnesses
that boot real services, agents, workflows, store-backed run history, plan/delegate,
and A2A with only the LLM mocked.

## Local and CI entry points

The default GitHub harness workflow runs this script on every push and pull
request after the install smoke check and 0→1 scaffold contract. Developers can
verify the installer seam alone with `make install-smoke`, or run the same
no-secret contract locally with:

```sh
make harness
```

That target intentionally exercises the install script smoke path, both 0→1
scaffold variants, the 0→hero scenario, the event-driven agent-flow harness, and
mock provider conformance, so
the public scaffold → run/chat → inspect → deploy lifecycle stays executable
outside CI as well. Live provider checks remain separate and gated by configured
API keys (`make provider-conformance` or the scheduled/manual CI job).
