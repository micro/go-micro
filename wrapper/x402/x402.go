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
//	    Price:   "10000",           // smallest units (e.g. 0.01 USDC)
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

// Config configures payment enforcement for a set of routes.
type Config struct {
	// PayTo is the address payments are sent to. Required.
	PayTo string
	// Network is the chain to settle on (default "base").
	Network string
	// Asset is the token contract/mint (default: the network's USDC).
	Asset string
	// Price is the amount required per request, in the asset's smallest
	// unit (e.g. "10000" for 0.01 USDC at 6 decimals).
	Price string
	// Description is shown to the paying client/agent.
	Description string
	// Facilitator verifies payments. Defaults to an HTTPFacilitator
	// pointed at FacilitatorURL.
	Facilitator Facilitator
	// FacilitatorURL is the verify/settle endpoint used when Facilitator
	// is nil (e.g. Coinbase CDP or Alchemy).
	FacilitatorURL string
}

func (c Config) network() string {
	if c.Network == "" {
		return "base"
	}
	return c.Network
}

func (c Config) requirements(r *http.Request) Requirements {
	return Requirements{
		Scheme:            "exact",
		Network:           c.network(),
		MaxAmountRequired: c.Price,
		Resource:          r.URL.Path,
		Description:       c.Description,
		PayTo:             c.PayTo,
		Asset:             c.Asset,
		MaxTimeoutSeconds: 60,
	}
}

// Middleware returns HTTP middleware that requires an x402 payment before
// the wrapped handler runs. A request without a valid X-PAYMENT header
// receives a 402 with the payment requirements; once a payment verifies,
// the request is served.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	fac := cfg.Facilitator
	if fac == nil {
		fac = &HTTPFacilitator{URL: cfg.FacilitatorURL}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			req := cfg.requirements(r)

			payment := r.Header.Get(PaymentHeader)
			if payment == "" {
				writeChallenge(w, req, "payment required")
				return
			}

			res, err := fac.Verify(r.Context(), payment, req)
			if err != nil {
				writeChallenge(w, req, "payment verification failed: "+err.Error())
				return
			}
			if !res.Valid {
				reason := res.Reason
				if reason == "" {
					reason = "payment invalid"
				}
				writeChallenge(w, req, reason)
				return
			}

			if res.Settlement != "" {
				w.Header().Set(PaymentResponseHeader, res.Settlement)
			}
			next.ServeHTTP(w, r)
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
