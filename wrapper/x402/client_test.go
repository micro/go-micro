package x402

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockPayer returns a fixed payment payload that the (mock) facilitator
// on the server accepts.
type mockPayer struct{ calls int }

func (p *mockPayer) Pay(ctx context.Context, req Requirements) (string, error) {
	p.calls++
	return "payment-ok", nil
}

// paidServer is an httptest server whose endpoint requires `amount`,
// verified by an always-valid facilitator.
func paidServer(amount string) *httptest.Server {
	cfg := Config{PayTo: "0xabc", Network: "base", Amount: amount, Facilitator: mockFacilitator{valid: true}}
	h := Middleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	return httptest.NewServer(h)
}

// The client pays a 402 within budget and gets the result; spend is tracked.
func TestClientPaysWithinBudget(t *testing.T) {
	srv := paidServer("10000")
	defer srv.Close()

	payer := &mockPayer{}
	c := &Client{Payer: payer, Budget: 50000}

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if b, _ := io.ReadAll(resp.Body); string(b) != "data" {
		t.Errorf("body = %q, want data", b)
	}
	if payer.calls != 1 {
		t.Errorf("payer called %d times, want 1", payer.calls)
	}
	if c.Spent() != 10000 {
		t.Errorf("spent = %d, want 10000", c.Spent())
	}
}

// A call that would exceed the budget is refused before paying.
func TestClientRefusesOverBudget(t *testing.T) {
	srv := paidServer("10000")
	defer srv.Close()

	payer := &mockPayer{}
	c := &Client{Payer: payer, Budget: 5000} // less than the price

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := c.Do(req)
	if err == nil {
		t.Fatal("expected an over-budget error")
	}
	if payer.calls != 0 {
		t.Errorf("payer should not be called when over budget (calls=%d)", payer.calls)
	}
	if c.Spent() != 0 {
		t.Errorf("nothing should be spent when refused, got %d", c.Spent())
	}
}

// The budget accumulates across calls and stops further spend.
func TestClientBudgetAccumulates(t *testing.T) {
	srv := paidServer("10000")
	defer srv.Close()

	c := &Client{Payer: &mockPayer{}, Budget: 15000}

	// First call (10000) fits.
	req1, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	if _, err := c.Do(req1); err != nil {
		t.Fatalf("first call: %v", err)
	}
	// Second call (another 10000) would total 20000 > 15000 — refused.
	req2, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	if _, err := c.Do(req2); err == nil {
		t.Fatal("second call should be refused (would exceed budget)")
	}
	if c.Spent() != 10000 {
		t.Errorf("spent = %d, want 10000", c.Spent())
	}
}

// A free endpoint needs no payer and no spend.
func TestClientFreeEndpoint(t *testing.T) {
	srv := paidServer("0") // free
	defer srv.Close()

	c := &Client{} // no payer
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 (free)", resp.StatusCode)
	}
	if c.Spent() != 0 {
		t.Errorf("free call should spend nothing, got %d", c.Spent())
	}
}
