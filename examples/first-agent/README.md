# First Agent

This is the smallest runnable service-backed agent in the repository. It sits
between `micro new helloworld` and the full [`examples/support`](../support/)
0→hero reference.

It runs with a deterministic mock model, so you do not need `ANTHROPIC_API_KEY`,
`OPENAI_API_KEY`, or any other provider secret.

```bash
go run ./examples/first-agent
```

Expected transcript:

```text
First agent (provider: mock, no API key)
> Summarize my next steps
  [notes] listed starter notes
assistant: Your first agent read the notes service and found three steps: install the CLI, run a service, then chat with an agent.
✓ service-backed agent completed without provider secrets
```

## What it demonstrates

- `notes` is a normal Go Micro service with one RPC method.
- `assistant` is an agent scoped to that service via `agent.Services("notes")`.
- The mock model requests the service tool through the normal agent tool handler.
- The final answer proves the service → agent path without a live model key.

CI keeps this path runnable with:

```bash
go test ./examples/first-agent
```

After this, continue to [`examples/support`](../support/) for the full services →
agents → workflows lifecycle with a flow trigger and an approval gate.
