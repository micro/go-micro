package x402

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	h := paidHandler(Config{PayTo: "0xabc", Network: "solana", Amount: "10000", Facilitator: mockFacilitator{valid: true}})

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
	h := paidHandler(Config{PayTo: "0xabc", Amount: "10000", Facilitator: mockFacilitator{valid: true}})

	r := httptest.NewRequest(http.MethodGet, "/tool", nil)
	r.Header.Set(PaymentHeader, "base64-payment-payload")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Header().Get(PaymentResponseHeader) != "0xtx" {
		t.Errorf("settlement not surfaced in %s", PaymentResponseHeader)
	}
}

// A payment the facilitator rejects gets a 402 with the reason.
func TestChallengeWhenPaymentInvalid(t *testing.T) {
	h := paidHandler(Config{PayTo: "0xabc", Amount: "10000", Facilitator: mockFacilitator{valid: false, reason: "insufficient amount"}})

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

// A free amount ("" or "0") requires no payment.
func TestRequireFreeAmount(t *testing.T) {
	cfg := Config{PayTo: "0xabc", Facilitator: mockFacilitator{valid: false}}
	rec := httptest.NewRecorder()
	if !cfg.Require(rec, httptest.NewRequest(http.MethodGet, "/free", nil), "0", "free.tool") {
		t.Error("a zero amount should be free and proceed")
	}
	// Require doesn't write on success; recorder defaults to 200.
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// Per-tool amounts override the default.
func TestAmountFor(t *testing.T) {
	cfg := Config{Amount: "100", Amounts: map[string]string{"paid.Tool.Do": "5000"}}
	if got := cfg.AmountFor("paid.Tool.Do"); got != "5000" {
		t.Errorf("per-tool amount = %q, want 5000", got)
	}
	if got := cfg.AmountFor("other.Tool.Do"); got != "100" {
		t.Errorf("default amount = %q, want 100", got)
	}
}

// LoadConfig reads an operator x402 config file.
func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x402.json")
	os.WriteFile(path, []byte(`{
		"payTo": "0xabc", "network": "solana", "asset": "USDC",
		"amount": "0", "amounts": {"weather.Weather.Forecast": "10000"}
	}`), 0o644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.PayTo != "0xabc" || cfg.Network != "solana" {
		t.Errorf("config not parsed: %+v", cfg)
	}
	if cfg.AmountFor("weather.Weather.Forecast") != "10000" {
		t.Errorf("per-tool amount not loaded: %+v", cfg.Amounts)
	}
	if cfg.AmountFor("anything.else") != "0" {
		t.Errorf("default amount = %q, want 0", cfg.AmountFor("anything.else"))
	}
}

// Network defaults to base when unset.
func TestNetworkDefault(t *testing.T) {
	if got := (Config{}).network(); got != "base" {
		t.Errorf("default network = %q, want base", got)
	}
}
