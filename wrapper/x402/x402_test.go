package x402

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockFacilitator struct {
	valid  bool
	reason string
}

func (m mockFacilitator) Verify(ctx context.Context, payment string, req Requirements) (Result, error) {
	return Result{Valid: m.valid, Reason: m.reason, Payer: "0xpayer", Settlement: "0xtx"}, nil
}

func paidHandler(cfg Config) http.Handler {
	served := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	return Middleware(cfg)(served)
}

// No payment yields a 402 with the requirements describing where to pay.
func TestChallengeWhenNoPayment(t *testing.T) {
	h := paidHandler(Config{PayTo: "0xabc", Network: "solana", Price: "10000", Facilitator: mockFacilitator{valid: true}})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tool", nil))

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", rec.Code)
	}
	var ch challenge
	if err := json.Unmarshal(rec.Body.Bytes(), &ch); err != nil {
		t.Fatalf("challenge body: %v", err)
	}
	if ch.X402Version != Version || len(ch.Accepts) != 1 {
		t.Fatalf("unexpected challenge: %+v", ch)
	}
	req := ch.Accepts[0]
	if req.PayTo != "0xabc" || req.Network != "solana" || req.MaxAmountRequired != "10000" {
		t.Errorf("requirements not advertised correctly: %+v", req)
	}
}

// A payment the facilitator accepts lets the request through, and the
// settlement is surfaced on the response.
func TestServesWhenPaymentValid(t *testing.T) {
	h := paidHandler(Config{PayTo: "0xabc", Price: "10000", Facilitator: mockFacilitator{valid: true}})

	r := httptest.NewRequest(http.MethodGet, "/tool", nil)
	r.Header.Set(PaymentHeader, "base64-payment-payload")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("handler not served, body = %q", rec.Body.String())
	}
	if rec.Header().Get(PaymentResponseHeader) != "0xtx" {
		t.Errorf("settlement not surfaced in %s", PaymentResponseHeader)
	}
}

// A payment the facilitator rejects gets a 402 with the reason.
func TestChallengeWhenPaymentInvalid(t *testing.T) {
	h := paidHandler(Config{PayTo: "0xabc", Price: "10000", Facilitator: mockFacilitator{valid: false, reason: "insufficient amount"}})

	r := httptest.NewRequest(http.MethodGet, "/tool", nil)
	r.Header.Set(PaymentHeader, "bad-payment")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", rec.Code)
	}
	var ch challenge
	json.Unmarshal(rec.Body.Bytes(), &ch)
	if ch.Error != "insufficient amount" {
		t.Errorf("reason not surfaced: %q", ch.Error)
	}
}

// Network defaults to base when unset.
func TestNetworkDefault(t *testing.T) {
	if got := (Config{}).network(); got != "base" {
		t.Errorf("default network = %q, want base", got)
	}
}
