# Restartable composition gate

This provider-free fixture is the executable v6 Stage 4 contract. It starts a
real RPC capability service and agent, runs bounded mock-model tool work inside
a checkpointed flow, durably waits for approval, and invokes an idempotent final
service operation. The flow host is replaced after every boundary while its
file-backed checkpoint is retained.

Run the release gate with:

```sh
make composition-gate
```

The mock path is the only CI requirement and never needs credentials. For an
optional live-provider composition check, run the existing agent-flow harness:

```sh
MICRO_AI_API_KEY=... go run ./internal/harness/agent-flow -provider openai
```

That live check is intentionally separate: provider availability and billing
must not determine whether restart, approval, lineage, or idempotency contracts
pass.
