package nano

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Errors a Verify call can return. Each one means "do not serve the request".
var (
	ErrNotConfirmed   = errors.New("nano: block is not a confirmed send")
	ErrWrongPayTo     = errors.New("nano: block does not pay the configured account")
	ErrWrongAmount    = errors.New("nano: block amount is not the exact price")
	ErrNotBound       = errors.New("nano: block representative does not commit to this order")
	ErrTooOld         = errors.New("nano: block is older than the payment window")
	ErrInvalidBlockID = errors.New("nano: invalid block hash")
)

// maxRPCResponse bounds how much of a node reply is read.
const maxRPCResponse = 64 << 10

// Verifier checks that a confirmed Nano send block pays exactly PriceRaw to
// PayTo for one order, then claims the block so it can never pay again. It
// reads the block from a node RPC the operator configures; it holds no key
// and no seed.
type Verifier struct {
	RPCURL   string
	PayTo    string
	PriceRaw *big.Int
	Claims   ClaimStore
	// MaxAge refuses blocks whose node timestamp is older than this. Zero
	// disables the window.
	MaxAge time.Duration
	Client *http.Client
	Now    func() time.Time
}

// Payment is what a successful Verify proves.
type Payment struct {
	BlockHash string
	Payer     string
	PayTo     string
	AmountRaw string
	OrderID   string
}

// NewVerifier validates the configuration: payTo must be a checksummed
// account, priceRaw positive, rpcURL https (plain http only on loopback),
// and claims non-nil.
func NewVerifier(rpcURL, payTo string, priceRaw *big.Int, claims ClaimStore) (*Verifier, error) {
	if err := ValidateAccount(payTo); err != nil {
		return nil, fmt.Errorf("nano: pay-to account: %w", err)
	}
	if priceRaw == nil || priceRaw.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}
	if err := checkRPCURL(rpcURL); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, errors.New("nano: a claim store is required")
	}
	return &Verifier{
		RPCURL:   rpcURL,
		PayTo:    payTo,
		PriceRaw: new(big.Int).Set(priceRaw),
		Claims:   claims,
		MaxAge:   15 * time.Minute,
		Client:   &http.Client{Timeout: 5 * time.Second},
		Now:      time.Now,
	}, nil
}

func checkRPCURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return errors.New("nano: invalid node RPC URL")
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		host := u.Hostname()
		if host == "localhost" {
			return nil
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return nil
		}
		return errors.New("nano: node RPC URL must be https unless it is loopback")
	default:
		return errors.New("nano: invalid node RPC URL scheme")
	}
}

// Offer returns the 402 entry for a new order.
func (v *Verifier) Offer(orderID string) Offer {
	price := v.PriceRaw.String()
	return Offer{
		Scheme:         Scheme,
		Network:        Network,
		Asset:          Asset,
		PayTo:          v.PayTo,
		Amount:         price,
		OrderID:        orderID,
		Representative: CommitmentAccount(orderID, v.PayTo, price),
	}
}

type blockInfo struct {
	Error          string `json:"error"`
	BlockAccount   string `json:"block_account"`
	Amount         string `json:"amount"`
	Confirmed      string `json:"confirmed"`
	Subtype        string `json:"subtype"`
	LocalTimestamp string `json:"local_timestamp"`
	Contents       struct {
		Type           string `json:"type"`
		Representative string `json:"representative"`
		LinkAsAccount  string `json:"link_as_account"`
	} `json:"contents"`
}

// Verify proves that blockHash pays this order and claims it. orderID is the
// payer's secret from the 402 challenge; a block hash alone is never enough.
func (v *Verifier) Verify(ctx context.Context, orderID, blockHash string) (*Payment, error) {
	return v.verify(ctx, orderID, blockHash, nil)
}

// verify is Verify with the price given per request (nil: the fixed PriceRaw).
func (v *Verifier) verify(ctx context.Context, orderID, blockHash string, priceRaw *big.Int) (*Payment, error) {
	want := v.priceOrFixed(priceRaw)
	if want == nil || want.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}
	if !ValidOrderID(orderID) {
		return nil, ErrInvalidOrderID
	}
	blockHash = strings.ToUpper(strings.TrimSpace(blockHash))
	if !blockHashPattern.MatchString(blockHash) {
		return nil, ErrInvalidBlockID
	}

	info, err := v.blockInfo(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	if info.Confirmed != "true" || info.Subtype != "send" || info.Contents.Type != "state" {
		return nil, ErrNotConfirmed
	}
	if !sameAccount(info.Contents.LinkAsAccount, v.PayTo) {
		return nil, ErrWrongPayTo
	}
	price := want.String()
	amount, err := ParseRaw(info.Amount)
	if err != nil || amount.Cmp(want) != 0 {
		return nil, ErrWrongAmount
	}
	if !sameAccount(info.Contents.Representative, CommitmentAccount(orderID, v.PayTo, price)) {
		return nil, ErrNotBound
	}
	if v.MaxAge > 0 {
		ts, err := strconv.ParseInt(info.LocalTimestamp, 10, 64)
		if err != nil || ts <= 0 || v.now().Sub(time.Unix(ts, 0)) > v.MaxAge {
			return nil, ErrTooOld
		}
	}
	if err := ValidateAccount(info.BlockAccount); err != nil {
		return nil, fmt.Errorf("nano: block account: %w", err)
	}
	if err := v.Claims.Claim(blockHash, orderID); err != nil {
		return nil, err
	}
	return &Payment{
		BlockHash: blockHash,
		Payer:     info.BlockAccount,
		PayTo:     v.PayTo,
		AmountRaw: price,
		OrderID:   orderID,
	}, nil
}

func (v *Verifier) now() time.Time {
	if v.Now != nil {
		return v.Now()
	}
	return time.Now()
}

func (v *Verifier) blockInfo(ctx context.Context, hash string) (*blockInfo, error) {
	body, _ := json.Marshal(map[string]string{"action": "block_info", "json_block": "true", "hash": hash})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.RPCURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := v.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nano: node RPC: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nano: node RPC status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxRPCResponse+1))
	if err != nil {
		return nil, fmt.Errorf("nano: node RPC: %w", err)
	}
	if len(raw) > maxRPCResponse {
		return nil, errors.New("nano: node RPC response too large")
	}
	var info blockInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("nano: node RPC: %w", err)
	}
	if info.Error != "" {
		return nil, ErrNotConfirmed
	}
	return &info, nil
}

func sameAccount(a, b string) bool {
	pa, err := DecodeAccount(a)
	if err != nil {
		return false
	}
	pb, err := DecodeAccount(b)
	if err != nil {
		return false
	}
	return bytes.Equal(pa, pb)
}
