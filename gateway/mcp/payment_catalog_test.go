package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v6/wrapper/x402"
)

// With payments enabled, /mcp/tools advertises each priced tool's payment
// requirements so the catalog is shoppable; free tools carry no payment.
func TestListToolsAdvertisesPayment(t *testing.T) {
	s := newTestServer(Options{
		Payment: &x402.Config{
			PayTo:   "0xabc",
			Network: "solana",
			Asset:   "USDC",
			Amount:  "0", // free by default
			Amounts: map[string]string{"weather.Weather.Forecast": "10000"},
		},
	})
	s.tools["weather.Weather.Forecast"] = &Tool{Name: "weather.Weather.Forecast", Description: "forecast"}
	s.tools["time.Time.Now"] = &Tool{Name: "time.Time.Now", Description: "now"}

	rec := httptest.NewRecorder()
	s.handleListTools(rec, httptest.NewRequest(http.MethodGet, "/mcp/tools", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out struct {
		Tools []struct {
			Name    string       `json:"name"`
			Payment *PaymentInfo `json:"payment"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}

	byName := map[string]*PaymentInfo{}
	for _, tl := range out.Tools {
		byName[tl.Name] = tl.Payment
	}

	paid := byName["weather.Weather.Forecast"]
	if paid == nil {
		t.Fatal("priced tool should advertise payment in the catalog")
	}
	if paid.Amount != "10000" || paid.Network != "solana" || paid.PayTo != "0xabc" || paid.Asset != "USDC" {
		t.Errorf("payment info wrong: %+v", paid)
	}
	if byName["time.Time.Now"] != nil {
		t.Error("free tool should not advertise payment")
	}
}

// Without payments configured, the catalog carries no payment info.
func TestListToolsNoPaymentWhenDisabled(t *testing.T) {
	s := newTestServer(Options{})
	s.tools["a.A.B"] = &Tool{Name: "a.A.B"}

	rec := httptest.NewRecorder()
	s.handleListTools(rec, httptest.NewRequest(http.MethodGet, "/mcp/tools", nil))

	var out struct {
		Tools []struct {
			Payment *PaymentInfo `json:"payment"`
		} `json:"tools"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Tools) != 1 || out.Tools[0].Payment != nil {
		t.Errorf("expected no payment info when payments disabled, got %+v", out.Tools)
	}
}
