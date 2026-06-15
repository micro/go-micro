// Package x402 implements the server side of the x402 payment protocol
// (the HTTP 402 "Payment Required" standard) as pluggable middleware.
//
// It lets a service or gateway require a stablecoin payment per request
// and verify it through a pluggable Facilitator (Coinbase CDP, Alchemy,
// or self-hosted), so AI agents can pay for tools and APIs autonomously.
// Go Micro stays chain-agnostic and free of crypto dependencies: it
// speaks the HTTP protocol and delegates verification and settlement to
// the facilitator, which does the on-chain work.
//
//	pay := x402.Middleware(x402.Config{
//	    PayTo:   "0xYourAddress",   // where payments go
//	    Network: "base",            // or "solana", ...
//	    Amount:  "10000",           // smallest units (e.g. 0.01 USDC)
//	})
//	mux.Handle("/paid", pay(handler))
//
// x402 is governed by the x402 Foundation (Linux Foundation). See
// https://x402.org and https://docs.cdp.coinbase.com/x402.
package x402

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// Version is the x402 protocol version this package speaks.
const Version = 1

// Header names defined by the protocol.
const (
	PaymentHeader         = "X-PAYMENT"          // request: the client's payment payload
	PaymentResponseHeader = "X-PAYMENT-RESPONSE" // response: settlement details
)

// Requirements describes what a client must pay to access a resource —
// the body of a 402 response (one entry of "accepts").
type Requirements struct {
	Scheme            string `json:"scheme"`                      // payment scheme, e.g. "exact"
	Network           string `json:"network"`                     // chain, e.g. "base", "solana"
	MaxAmountRequired string `json:"maxAmountRequired"`           // amount in the asset's smallest unit
	Resource          string `json:"resource"`                    // the resource being paid for
	Description       string `json:"description,omitempty"`       // human/agent-readable description
	PayTo             string `json:"payTo"`                       // receiving address
	Asset             string `json:"asset,omitempty"`             // token contract/mint (default: network USDC)
	MaxTimeoutSeconds int    `json:"maxTimeoutSeconds,omitempty"` // how long the client has to pay
}

// challenge is the JSON body returned with a 402 response.
type challenge struct {
	X402Version int            `json:"x402Version"`
	Accepts     []Requirements `json:"accepts"`
	Error       string         `json:"error,omitempty"`
}

// Result is the outcome of verifying a payment.
type Result struct {
	Valid      bool   // whether the payment satisfies the requirements
	Payer      string // the paying address, if known
	Reason     string // why the payment was rejected, if not valid
	Settlement string // settlement reference (e.g. tx hash), set into X-PAYMENT-RESPONSE
}

// Facilitator verifies (and optionally settles) a payment a client
// presented against the stated requirements. Implementations talk to a
// chain or a hosted facilitator; the gateway stays chain-agnostic, so a
// Base facilitator and a Solana facilitator are just different
// implementations behind this interface.
type Facilitator interface {
	Verify(ctx context.Context, payment string, req Requirements) (Result, error)
}

// Config configures payment enforcement for a set of routes or tools.
type Config struct {
	// PayTo is the address payments are sent to. Required.
	PayTo string `json:"payTo"`
	// Network is the chain to settle on (default "base").
	Network string `json:"network,omitempty"`
	// Asset is the token contract/mint (default: the network's USDC).
	Asset string `json:"asset,omitempty"`
	// Amount is the default amount required per request, in the asset's
	// smallest unit (e.g. "10000" for 0.01 USDC at 6 decimals). "0" or
	// empty means free.
	Amount string `json:"amount,omitempty"`
	// Amounts overrides Amount per tool/resource name, so an operator can
	// charge for tools individually — the way Scopes and RateLimit are
	// configured per tool at the gateway.
	Amounts map[string]string `json:"amounts,omitempty"`
	// Description is shown to the paying client/agent.
	Description string `json:"description,omitempty"`
	// Facilitator verifies payments. Defaults to an HTTPFacilitator
	// pointed at FacilitatorURL.
	Facilitator Facilitator `json:"-"`
	// FacilitatorURL is the verify/settle endpoint used when Facilitator
	// is nil (e.g. Coinbase CDP or Alchemy).
	FacilitatorURL string `json:"facilitator,omitempty"`
}

func (c Config) network() string {
	if c.Network == "" {
		return "base"
	}
	return c.Network
}

// AmountFor returns the amount required for a named tool/resource: the
// per-tool override if present, otherwise the default Amount.
func (c Config) AmountFor(name string) string {
	if a, ok := c.Amounts[name]; ok {
		return a
	}
	return c.Amount
}

func (c Config) facilitator() Facilitator {
	if c.Facilitator != nil {
		return c.Facilitator
	}
	return &HTTPFacilitator{URL: c.FacilitatorURL}
}

func (c Config) requirements(amount, resource string) Requirements {
	return Requirements{
		Scheme:            "exact",
		Network:           c.network(),
		MaxAmountRequired: amount,
		Resource:          resource,
		Description:       c.Description,
		PayTo:             c.PayTo,
		Asset:             c.Asset,
		MaxTimeoutSeconds: 60,
	}
}

// Require enforces payment of amount for a single request. It returns
// true if the request may proceed — the amount is free ("" or "0"), or a
// valid payment was presented — and false once it has written a 402
// challenge, in which case the caller must stop. resource names what is
// being paid for (a tool name or URL path).
func (c Config) Require(w http.ResponseWriter, r *http.Request, amount, resource string) bool {
	if amount == "" || amount == "0" {
		return true // free
	}
	req := c.requirements(amount, resource)

	payment := r.Header.Get(PaymentHeader)
	if payment == "" {
		writeChallenge(w, req, "payment required")
		return false
	}
	res, err := c.facilitator().Verify(r.Context(), payment, req)
	if err != nil {
		writeChallenge(w, req, "payment verification failed: "+err.Error())
		return false
	}
	if !res.Valid {
		reason := res.Reason
		if reason == "" {
			reason = "payment invalid"
		}
		writeChallenge(w, req, reason)
		return false
	}
	if res.Settlement != "" {
		w.Header().Set(PaymentResponseHeader, res.Settlement)
	}
	return true
}

// Middleware returns HTTP middleware that requires the default Amount for
// any wrapped route. For per-tool amounts, resolve the amount with
// AmountFor and call Require directly (the MCP gateway does this).
func Middleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Require(w, r, cfg.Amount, r.URL.Path) {
				next.ServeHTTP(w, r)
			}
		})
	}
}

func writeChallenge(w http.ResponseWriter, req Requirements, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired) // 402
	_ = json.NewEncoder(w).Encode(challenge{
		X402Version: Version,
		Accepts:     []Requirements{req},
		Error:       reason,
	})
}

// LoadConfig reads an x402 config file (JSON) describing the operator's
// payTo address, network, asset, default amount, and per-tool amounts:
//
//	{ "payTo": "0x…", "network": "solana", "asset": "USDC",
//	  "amount": "0", "amounts": { "weather.Weather.Forecast": "10000" } }
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse x402 config %s: %w", path, err)
	}
	return &c, nil
}

// HTTPFacilitator verifies payments by POSTing to an x402 facilitator's
// verify endpoint (Coinbase CDP, Alchemy, or self-hosted). It carries no
// chain or crypto code itself.
type HTTPFacilitator struct {
	URL    string
	Client *http.Client
}

func (f *HTTPFacilitator) Verify(ctx context.Context, payment string, req Requirements) (Result, error) {
	if f.URL == "" {
		return Result{}, fmt.Errorf("no facilitator configured")
	}
	body, _ := json.Marshal(map[string]any{
		"x402Version":         Version,
		"paymentPayload":      payment,
		"paymentRequirements": req,
	})
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, f.URL+"/verify", bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	hreq.Header.Set("Content-Type", "application/json")
	cl := f.Client
	if cl == nil {
		cl = http.DefaultClient
	}
	resp, err := cl.Do(hreq)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	var out struct {
		IsValid       bool   `json:"isValid"`
		InvalidReason string `json:"invalidReason"`
		Payer         string `json:"payer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Result{}, err
	}
	return Result{Valid: out.IsValid, Reason: out.InvalidReason, Payer: out.Payer}, nil
}
