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
// For real settlement on Base mainnet, point it at the Coinbase CDP
// facilitator, which requires an authenticated request:
//
//	cfg.Facilitator = x402.CDP(os.Getenv("CDP_API_KEY_ID"), os.Getenv("CDP_API_KEY_SECRET"))
//
// x402 is governed by the x402 Foundation (Linux Foundation). See
// https://x402.org and https://docs.cdp.coinbase.com/x402.
package x402

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// Version is the x402 protocol version this package speaks.
const Version = 1

// Header names defined by the protocol. Version 1 uses X-PAYMENT /
// X-PAYMENT-RESPONSE; version 2 renamed them to PAYMENT-SIGNATURE /
// PAYMENT-RESPONSE. We accept either request header and emit both response
// headers so any conformant client interoperates.
const (
	PaymentHeader           = "X-PAYMENT"          // request: the client's payment payload
	PaymentHeaderV2         = "PAYMENT-SIGNATURE"  // request: v2 alias
	PaymentResponseHeader   = "X-PAYMENT-RESPONSE" // response: settlement details
	PaymentResponseHeaderV2 = "PAYMENT-RESPONSE"   // response: v2 alias
)

// Requirements describes what a client must pay to access a resource —
// the body of a 402 response (one entry of "accepts").
type Requirements struct {
	Scheme            string            `json:"scheme"`                      // payment scheme, e.g. "exact"
	Network           string            `json:"network"`                     // chain, e.g. "base", "eip155:8453"
	MaxAmountRequired string            `json:"maxAmountRequired"`           // amount in the asset's smallest unit
	Resource          string            `json:"resource"`                    // the resource being paid for
	Description       string            `json:"description,omitempty"`       // human/agent-readable description
	MimeType          string            `json:"mimeType,omitempty"`          // response mime type
	PayTo             string            `json:"payTo"`                       // receiving address
	Asset             string            `json:"asset,omitempty"`             // token contract/mint (default: network USDC)
	MaxTimeoutSeconds int               `json:"maxTimeoutSeconds,omitempty"` // how long the client has to pay
	Extra             map[string]string `json:"extra,omitempty"`             // scheme extras, e.g. EIP-712 domain {name, version}
}

// challenge is the JSON body returned with a 402 response.
type challenge struct {
	X402Version int            `json:"x402Version"`
	Accepts     []Requirements `json:"accepts"`
	Error       string         `json:"error,omitempty"`
}

// Result is the outcome of verifying (or settling) a payment.
type Result struct {
	Valid      bool   // whether the payment satisfies the requirements
	Payer      string // the paying address, if known
	Reason     string // why the payment was rejected, if not valid
	Settlement string // settlement reference (e.g. tx hash), set into X-PAYMENT-RESPONSE
}

// Facilitator verifies a payment a client presented against the stated
// requirements. Implementations talk to a chain or a hosted facilitator; the
// gateway stays chain-agnostic, so a Base facilitator and a Solana facilitator
// are just different implementations behind this interface.
type Facilitator interface {
	Verify(ctx context.Context, payment string, req Requirements) (Result, error)
}

// Settler is an optional capability: a Facilitator that also settles a verified
// payment on-chain (captures the funds) and returns a settlement reference.
// Require calls Settle after a successful Verify when the facilitator
// implements it — for the "exact" scheme, verify authorizes and settle
// captures, so without settlement no funds actually move.
type Settler interface {
	Settle(ctx context.Context, payment string, req Requirements) (Result, error)
}

// Config configures payment enforcement for a set of routes or tools.
type Config struct {
	// PayTo is the address payments are sent to. Required.
	PayTo string `json:"payTo"`
	// Network is the chain to settle on (default "base"). Accepts short
	// names ("base", "base-sepolia") or CAIP-2 ids ("eip155:8453").
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
	// Extra carries scheme-specific data echoed in the requirement's "extra".
	// For the EVM "exact" scheme this is the asset's EIP-712 domain, e.g.
	// {"name":"USD Coin","version":"2"}; when empty it is filled in for known
	// assets so clients can build a valid transfer signature.
	Extra map[string]string `json:"extra,omitempty"`
	// Facilitator verifies (and, if it implements Settler, settles) payments.
	// Defaults to an HTTPFacilitator pointed at FacilitatorURL.
	Facilitator Facilitator `json:"-"`
	// FacilitatorURL is the verify/settle endpoint used when Facilitator
	// is nil (e.g. Coinbase CDP or Alchemy).
	FacilitatorURL string `json:"facilitator,omitempty"`
	// RequireSettlement fails closed when a paid request cannot be settled:
	// if the facilitator only verifies (does not implement Settler), Require
	// refuses to serve rather than releasing the resource while no funds move.
	// Leave false only for verify-only flows where authorization is enough.
	RequireSettlement bool `json:"requireSettlement,omitempty"`
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
	net := c.network()
	asset := c.Asset
	extra := c.Extra
	// Fill the asset and its EIP-712 domain for known networks so a client
	// can sign without the operator hand-configuring token metadata. Keyed on
	// the CAIP-2 form so both "base" and "eip155:8453" resolve.
	if def, ok := defaultAsset(NormalizeNetwork(net)); ok {
		if asset == "" {
			asset = def.Address
		}
		if extra == nil && strings.EqualFold(asset, def.Address) {
			extra = map[string]string{"name": def.Name, "version": def.Version}
		}
	}
	return Requirements{
		Scheme:            "exact",
		Network:           net,
		MaxAmountRequired: amount,
		Resource:          resource,
		Description:       c.Description,
		MimeType:          "application/json",
		PayTo:             c.PayTo,
		Asset:             asset,
		MaxTimeoutSeconds: 60,
		Extra:             extra,
	}
}

// Payment returns the client's payment payload from either the v1 or v2
// request header, or "" if none is present.
func Payment(r *http.Request) string {
	if p := r.Header.Get(PaymentHeader); p != "" {
		return p
	}
	return r.Header.Get(PaymentHeaderV2)
}

// Require enforces payment of amount for a single request. It returns
// true if the request may proceed — the amount is free ("" or "0"), or a
// valid payment was presented (and settled, when the facilitator supports
// it) — and false once it has written a 402 challenge, in which case the
// caller must stop. resource names what is being paid for (a tool name or
// URL path).
func (c Config) Require(w http.ResponseWriter, r *http.Request, amount, resource string) bool {
	if amount == "" || amount == "0" {
		return true // free
	}
	req := c.requirements(amount, resource)

	payment := Payment(r)
	if payment == "" {
		writeChallenge(w, req, "payment required")
		return false
	}
	fac := c.facilitator()
	res, err := fac.Verify(r.Context(), payment, req)
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
	// Capture the funds when the facilitator can settle. Verify alone only
	// authorizes the "exact" transfer; settlement broadcasts it.
	s, canSettle := fac.(Settler)
	if c.RequireSettlement && !canSettle {
		// Fail closed: a paid config must not serve the resource on a
		// verify-only facilitator, or it gives the tool away for free.
		writeChallenge(w, req, "payment settlement unavailable")
		return false
	}
	if canSettle {
		sres, err := s.Settle(r.Context(), payment, req)
		if err != nil {
			writeChallenge(w, req, "payment settlement failed: "+err.Error())
			return false
		}
		if !sres.Valid {
			reason := sres.Reason
			if reason == "" {
				reason = "settlement rejected"
			}
			writeChallenge(w, req, reason)
			return false
		}
		if sres.Settlement != "" {
			res.Settlement = sres.Settlement
		}
	}
	if res.Settlement != "" {
		setSettlementHeaders(w, res.Settlement)
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

func setSettlementHeaders(w http.ResponseWriter, settlement string) {
	w.Header().Set(PaymentResponseHeader, settlement)
	w.Header().Set(PaymentResponseHeaderV2, settlement)
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

// HTTPFacilitator verifies and settles payments by POSTing to an x402
// facilitator's /verify and /settle endpoints (Coinbase CDP, Alchemy, or
// self-hosted). It carries no chain or crypto code itself; when the endpoint
// requires authentication (e.g. CDP), set Authorize to attach credentials —
// see CDP.
type HTTPFacilitator struct {
	URL    string
	Client *http.Client
	// Authorize, when set, is called on each facilitator request to attach
	// authentication (e.g. a Bearer token). Nil for open facilitators.
	Authorize func(*http.Request) error
}

// Verify checks the payment is valid against the requirements.
func (f *HTTPFacilitator) Verify(ctx context.Context, payment string, req Requirements) (Result, error) {
	out, err := f.post(ctx, "/verify", payment, req)
	if err != nil {
		return Result{}, err
	}
	return Result{Valid: out.IsValid, Reason: firstNonEmpty(out.InvalidReason, out.Error), Payer: out.Payer}, nil
}

// Settle captures a verified payment on-chain and returns the transaction
// reference, satisfying Settler.
func (f *HTTPFacilitator) Settle(ctx context.Context, payment string, req Requirements) (Result, error) {
	out, err := f.post(ctx, "/settle", payment, req)
	if err != nil {
		return Result{}, err
	}
	// Facilitators report settlement as "success" with a "transaction" ref.
	ok := out.Success || out.IsValid
	return Result{Valid: ok, Reason: firstNonEmpty(out.ErrorReason, out.InvalidReason, out.Error), Payer: out.Payer, Settlement: out.Transaction}, nil
}

type facilitatorResponse struct {
	IsValid       bool   `json:"isValid"`
	Success       bool   `json:"success"`
	InvalidReason string `json:"invalidReason"`
	ErrorReason   string `json:"errorReason"`
	Error         string `json:"error"`
	Payer         string `json:"payer"`
	Transaction   string `json:"transaction"`
}

func (f *HTTPFacilitator) post(ctx context.Context, path, payment string, req Requirements) (facilitatorResponse, error) {
	var out facilitatorResponse
	if f.URL == "" {
		return out, fmt.Errorf("no facilitator configured")
	}
	body, _ := json.Marshal(map[string]any{
		"x402Version":         Version,
		"paymentPayload":      decodePayment(payment),
		"paymentRequirements": req,
	})
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(f.URL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	hreq.Header.Set("Content-Type", "application/json")
	if f.Authorize != nil {
		if err := f.Authorize(hreq); err != nil {
			return out, fmt.Errorf("authorize: %w", err)
		}
	}
	cl := f.Client
	if cl == nil {
		cl = http.DefaultClient
	}
	resp, err := cl.Do(hreq)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return out, fmt.Errorf("facilitator %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(buf.String()))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	return out, nil
}

// decodePayment turns the base64 X-PAYMENT header into the PaymentPayload
// object facilitators expect. If it is not base64 JSON, the raw value is passed
// through unchanged (some facilitators accept the encoded string).
func decodePayment(payment string) any {
	payment = strings.TrimSpace(payment)
	for _, dec := range []*base64.Encoding{base64.StdEncoding, base64.RawURLEncoding} {
		if raw, err := dec.DecodeString(payment); err == nil {
			var obj any
			if json.Unmarshal(raw, &obj) == nil {
				return obj
			}
		}
	}
	return payment
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
