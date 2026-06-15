package x402

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
)

// Payer produces a payment payload for the given requirements. A real
// implementation signs a stablecoin authorization with a wallet; tests
// and local development can return a fixed token. It is the consumer
// counterpart to the Facilitator (which verifies on the server side).
type Payer interface {
	Pay(ctx context.Context, req Requirements) (payment string, err error)
}

// Client is an HTTP client that automatically settles x402 payment
// challenges, up to an optional spend Budget. It is the consumer
// counterpart to Middleware: point it at a paid tool, and a 402 is paid
// and retried transparently — unless paying would exceed the budget, the
// spend cap that keeps an autonomous, paying caller in bounds.
type Client struct {
	// HTTP is the underlying client (defaults to http.DefaultClient).
	HTTP *http.Client
	// Payer constructs payment payloads. Required to pay.
	Payer Payer
	// Budget caps total spend across all calls, in the asset's smallest
	// unit (0 = unlimited). A call that would exceed it is refused before
	// any payment is made.
	Budget int64

	mu    sync.Mutex
	spent int64
}

// Spent returns the total amount paid so far, in the asset's smallest unit.
func (c *Client) Spent() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.spent
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// Do performs req. If the server answers 402, Do reads the requirements,
// checks the budget, pays via Payer, and retries once with the payment
// attached. It returns an error (without paying) if the payment would
// exceed the budget.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// Buffer the body so the request can be replayed after paying.
	var body []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
		body = b
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	resp, err := c.httpClient().Do(req)
	if err != nil || resp.StatusCode != http.StatusPaymentRequired {
		return resp, err
	}

	var ch challenge
	_ = json.NewDecoder(resp.Body).Decode(&ch)
	resp.Body.Close()
	if len(ch.Accepts) == 0 {
		return resp, fmt.Errorf("x402: 402 response carried no requirements")
	}
	reqd := ch.Accepts[0]
	amount, _ := strconv.ParseInt(reqd.MaxAmountRequired, 10, 64)

	// Spend cap: refuse before paying if this would exceed the budget.
	c.mu.Lock()
	if c.Budget > 0 && c.spent+amount > c.Budget {
		spent := c.spent
		c.mu.Unlock()
		return nil, fmt.Errorf("x402: paying %d for %s would exceed budget (spent %d of %d)",
			amount, reqd.Resource, spent, c.Budget)
	}
	c.mu.Unlock()

	if c.Payer == nil {
		return nil, fmt.Errorf("x402: payment required for %s but no Payer configured", reqd.Resource)
	}
	payment, err := c.Payer.Pay(req.Context(), reqd)
	if err != nil {
		return nil, fmt.Errorf("x402: pay: %w", err)
	}

	// Replay the request with the payment attached.
	var rbody io.Reader
	if body != nil {
		rbody = bytes.NewReader(body)
	}
	retry, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), rbody)
	if err != nil {
		return nil, err
	}
	for k, v := range req.Header {
		retry.Header[k] = v
	}
	retry.Header.Set(PaymentHeader, payment)

	resp2, err := c.httpClient().Do(retry)
	if err != nil {
		return nil, err
	}
	if resp2.StatusCode == http.StatusPaymentRequired {
		return resp2, fmt.Errorf("x402: payment for %s was rejected", reqd.Resource)
	}

	c.mu.Lock()
	c.spent += amount
	c.mu.Unlock()
	return resp2, nil
}
