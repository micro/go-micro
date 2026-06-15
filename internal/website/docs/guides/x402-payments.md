---
layout: default
---

# Payments (x402)

Go Micro can require a payment before a tool runs, using [x402](https://x402.org) — the open HTTP **402 Payment Required** standard for stablecoin payments, designed for AI agents and onchain APIs. It lets every Go Micro endpoint, already exposed as an AI-callable tool, become a *paid* tool: a service answers a call with `402` and payment requirements, the client pays and retries, and the gateway verifies the payment before serving.

Payments are **opt-in** and **dependency-light**. Go Micro carries no chain or crypto code — it speaks the protocol and delegates verification and settlement to a pluggable **facilitator** (Coinbase CDP, Alchemy, or self-hosted), so Base and Solana are just different facilitators behind one interface.

## The wrapper

The core is HTTP middleware in `go-micro.dev/v5/wrapper/x402`:

```go
import "go-micro.dev/v5/wrapper/x402"

pay := x402.Middleware(x402.Config{
    PayTo:          "0xYourAddress",            // where payments go (required)
    Network:        "base",                     // or "solana", ...
    Amount:         "10000",                    // smallest units, e.g. 0.01 USDC
    FacilitatorURL: "https://facilitator.example",
})
mux.Handle("/paid", pay(handler))
```

A request with no `X-PAYMENT` header gets a `402` with the requirements; once a payment verifies through the facilitator, the request is served (with settlement details on the `X-PAYMENT-RESPONSE` header).

### Pluggable facilitator

`Config.Facilitator` is an interface; the default is an `HTTPFacilitator` pointed at `FacilitatorURL`. Implement your own to target any chain or hosted service:

```go
type Facilitator interface {
    Verify(ctx context.Context, payment string, req Requirements) (Result, error)
}
```

## At the MCP gateway

Because every endpoint is already an MCP tool, the gateway is where you charge. Payments are wired into both `micro mcp serve` and the standalone `micro-mcp-gateway`, gated on `/mcp/call` (listing tools and health stay free), and **off unless you set a pay-to address**.

```bash
micro mcp serve --address :3000 \
    --x402-pay-to 0xYourAddress \
    --x402-network solana \
    --x402-amount 10000 \
    --x402-facilitator https://facilitator.example
```

## Per-tool amounts

Different tools can cost different amounts. Pricing is an **operator** concern — the payTo address is the operator's, and amounts change without redeploying anyone's service — so it's configured at the gateway with a file, the same way per-tool scopes and rate limits are. Point the gateway at an x402 config:

```bash
micro mcp serve --address :3000 --x402-config x402.json
```

```json
{
  "payTo": "0xYourAddress",
  "network": "solana",
  "asset": "USDC",
  "amount": "0",
  "amounts": {
    "weather.Weather.Forecast": "10000",
    "search.Search.Query": "5000"
  }
}
```

`amount` is the default (here `"0"` — free unless priced), and `amounts` sets per-tool overrides keyed by tool name. There is no "pricing" abstraction; it's the x402 `amount`, resolved per tool, in the protocol's own vocabulary. The standalone gateway accepts the same file via `--x402-config` or the `X402_CONFIG` environment variable.

## Notes

- **Opt-in.** No pay-to address (and no config), no payments — nothing changes.
- **No crypto in the framework.** The facilitator does verification and settlement on-chain; Go Micro speaks HTTP.
- **A paying agent needs a budget.** On the agent side, an unattended agent that spends money needs a spend cap next to `MaxSteps` and `ApproveTool` — see [Plan & Delegate](plan-delegate.html) for the guardrail model. This is active work.

## See also

- [Building Effective Agents — Agents and Workflows](agents-and-workflows.html)
- [MCP & AI Agents](../mcp.html)
- [x402 — Coinbase Developer Docs](https://docs.cdp.coinbase.com/x402/welcome) · [x402 on Solana](https://solana.com/x402/what-is-x402)
