package nano

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v6/wrapper/x402"
)

// Facilitator verifies nano:mainnet payments with scheme "exact" for
// x402.Config. It is a self-hosted facilitator: it reads the paying block from
// a Nano node RPC the operator configures and holds no key and no seed.
//
//	claims, _ := nano.OpenFileClaimStore("/var/lib/myservice/nano-claims")
//	fac, _ := nano.NewFacilitator("https://node.example/rpc", "nano_1...", claims)
//	price, _ := nano.RawFromXNO("0.001") // 1 XNO = 10^30 raw
//	pay := x402.Middleware(x402.Config{
//	    PayTo:       "nano_1...",
//	    Network:     "nano",
//	    Amount:      price.String(),
//	    Facilitator: fac,
//	})
//
// Each 402 carries a fresh 128-bit order id in "extra". The payer sends
// exactly the price to payTo with the block's representative set to
// extra.representative, a commitment to (order id, payTo, price), and then
// presents {"orderId", "block"} as the payment payload. A block pays for one
// order, once: the claim is written to the ClaimStore before the request is
// served, and a FileClaimStore keeps it across restarts.
type Facilitator struct {
	v *Verifier

	mu       sync.Mutex
	verified map[string]string // block hash -> order id, verified and claimed, not yet settled
}

// NewFacilitator validates the configuration: payTo must be a checksummed
// account, rpcURL https (plain http only on loopback), and claims non-nil.
func NewFacilitator(rpcURL, payTo string, claims ClaimStore) (*Facilitator, error) {
	if err := ValidateAccount(payTo); err != nil {
		return nil, fmt.Errorf("nano: pay-to account: %w", err)
	}
	if err := checkRPCURL(rpcURL); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, errors.New("nano: a claim store is required")
	}
	return &Facilitator{
		v: &Verifier{
			RPCURL: rpcURL,
			PayTo:  payTo,
			Claims: claims,
			MaxAge: 15 * time.Minute,
			Client: &http.Client{Timeout: 5 * time.Second},
			Now:    time.Now,
		},
		verified: map[string]string{},
	}, nil
}

// Verifier exposes the underlying verifier so the operator can change its
// HTTP client or payment window.
func (f *Facilitator) Verifier() *Verifier { return f.v }

// Challenge implements x402.Challenger: it adds a fresh order id and the
// representative that commits to it. Requirements for another network are
// returned unchanged.
func (f *Facilitator) Challenge(req x402.Requirements) x402.Requirements {
	if x402.NormalizeNetwork(req.Network) != Network {
		return req
	}
	price, err := ParseRaw(req.MaxAmountRequired)
	if err != nil || price.Sign() <= 0 {
		return req
	}
	orderID, err := NewOrderID()
	if err != nil {
		return req
	}
	extra := make(map[string]string, len(req.Extra)+2)
	for k, v := range req.Extra {
		extra[k] = v
	}
	extra["orderId"] = orderID
	extra["representative"] = CommitmentAccount(orderID, req.PayTo, price.String())
	req.Network = Network
	req.Scheme = Scheme
	if req.Asset == "" {
		req.Asset = Asset
	}
	req.Extra = extra
	return req
}

// payload is the decoded payment header.
type payload struct {
	Scheme  string `json:"scheme"`
	Network string `json:"network"`
	Payload struct {
		OrderID string `json:"orderId"`
		Block   string `json:"block"`
	} `json:"payload"`
}

func decodePayload(payment string) (*payload, error) {
	payment = strings.TrimSpace(payment)
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		raw, err := enc.DecodeString(payment)
		if err != nil {
			continue
		}
		var p payload
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, errInvalidPayload
		}
		if p.Scheme != Scheme || x402.NormalizeNetwork(p.Network) != Network {
			return nil, errInvalidPayload
		}
		return &p, nil
	}
	return nil, errInvalidPayload
}

var errInvalidPayload = errors.New("nano: payment payload is not a nano:mainnet exact payment")

// Verify implements x402.Facilitator. A valid result means the block is a
// confirmed send of exactly req.MaxAmountRequired raw to req.PayTo, bound to
// the presented order id, and has just been claimed.
func (f *Facilitator) Verify(ctx context.Context, payment string, req x402.Requirements) (x402.Result, error) {
	if x402.NormalizeNetwork(req.Network) != Network {
		return x402.Result{Reason: "nano: requirements are not for nano:mainnet"}, nil
	}
	if !sameAccount(req.PayTo, f.v.PayTo) {
		return x402.Result{Reason: "nano: requirements pay-to is not this facilitator's account"}, nil
	}
	price, err := ParseRaw(req.MaxAmountRequired)
	if err != nil || price.Sign() <= 0 {
		return x402.Result{Reason: "nano: requirements amount is not a positive raw amount"}, nil
	}
	p, err := decodePayload(payment)
	if err != nil {
		return x402.Result{Reason: err.Error()}, nil
	}
	paid, err := f.v.verify(ctx, p.Payload.OrderID, p.Payload.Block, price)
	if err != nil {
		if isRefusal(err) {
			return x402.Result{Reason: err.Error()}, nil
		}
		return x402.Result{}, err
	}
	f.mu.Lock()
	f.verified[paid.BlockHash] = paid.OrderID
	f.mu.Unlock()
	return x402.Result{Valid: true, Payer: paid.Payer, Settlement: paid.BlockHash}, nil
}

// Settle implements x402.Settler. A confirmed Nano send is already final, so
// settlement only hands back the block hash of a payment this facilitator
// verified and has not settled yet; anything else is refused.
func (f *Facilitator) Settle(ctx context.Context, payment string, req x402.Requirements) (x402.Result, error) {
	p, err := decodePayload(payment)
	if err != nil {
		return x402.Result{Reason: err.Error()}, nil
	}
	hash := strings.ToUpper(strings.TrimSpace(p.Payload.Block))
	f.mu.Lock()
	order, ok := f.verified[hash]
	if ok && order == p.Payload.OrderID {
		delete(f.verified, hash)
	}
	f.mu.Unlock()
	if !ok || order != p.Payload.OrderID {
		return x402.Result{Reason: "nano: payment was not verified by this facilitator"}, nil
	}
	return x402.Result{Valid: true, Settlement: hash}, nil
}

func isRefusal(err error) bool {
	for _, e := range []error{
		ErrAlreadyClaimed, ErrInvalidOrderID, ErrInvalidBlockID, ErrNotConfirmed,
		ErrWrongPayTo, ErrWrongAmount, ErrNotBound, ErrTooOld, ErrInvalidAccount,
	} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

// priceOrFixed returns price when set, else the verifier's fixed price.
func (v *Verifier) priceOrFixed(price *big.Int) *big.Int {
	if price != nil {
		return price
	}
	return v.PriceRaw
}
