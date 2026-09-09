# Agent x402 buyer

This example shows an agent paying for a paid HTTP tool with x402 without using
live funds or a live chain.

It starts a local paid endpoint guarded by `wrapper/x402` seller middleware and a
mock facilitator. A deterministic mock-model agent calls that endpoint as a tool,
receives the HTTP 402 challenge, pays with `AgentPayer`, stays inside
`AgentBudget`, retries the request, and prints the spend recorded for the run.

```bash
go run ./examples/agent-x402-buyer
```

Expected output includes:

- the paid tool response,
- one facilitator verify and settle call, and
- `run spend: 7 smallest units (budget 10)`.

The payment token and facilitator are intentionally local development fakes. To
settle real x402 payments, keep the same `AgentPayer` / `AgentBudget` shape but
replace the payer with a wallet-backed implementation and configure the seller
middleware with a hosted or self-run x402 facilitator.
