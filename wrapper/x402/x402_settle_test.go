package x402

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestVerifyAndSettle checks that Require verifies then settles against an
// HTTP facilitator and surfaces the settlement in both response headers.
func TestVerifyAndSettle(t *testing.T) {
	var verifyHit, settleHit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/verify":
			verifyHit = true
			_ = json.NewEncoder(w).Encode(map[string]any{"isValid": true, "payer": "0xabc"})
		case "/settle":
			settleHit = true
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "transaction": "0xdeadbeef"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := Config{PayTo: "0xpay", Network: "eip155:8453", FacilitatorURL: srv.URL}
	r := httptest.NewRequest(http.MethodGet, "/tool", nil)
	r.Header.Set(PaymentHeader, base64.StdEncoding.EncodeToString([]byte(`{"network":"eip155:8453"}`)))
	rec := httptest.NewRecorder()

	if !cfg.Require(rec, r, "10000", "chat") {
		t.Fatalf("Require returned false; body=%s", rec.Body.String())
	}
	if !verifyHit || !settleHit {
		t.Fatalf("expected both verify and settle to be hit: verify=%v settle=%v", verifyHit, settleHit)
	}
	if got := rec.Header().Get(PaymentResponseHeader); got != "0xdeadbeef" {
		t.Errorf("X-PAYMENT-RESPONSE = %q, want 0xdeadbeef", got)
	}
	if got := rec.Header().Get(PaymentResponseHeaderV2); got != "0xdeadbeef" {
		t.Errorf("PAYMENT-RESPONSE = %q, want 0xdeadbeef", got)
	}
}

// TestSettlementFailureChallenges checks that a failed settlement blocks the
// request with a fresh 402 rather than letting it through.
func TestSettlementFailureChallenges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/verify":
			_ = json.NewEncoder(w).Encode(map[string]any{"isValid": true})
		case "/settle":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "errorReason": "insufficient_funds"})
		}
	}))
	defer srv.Close()

	cfg := Config{PayTo: "0xpay", FacilitatorURL: srv.URL}
	r := httptest.NewRequest(http.MethodGet, "/tool", nil)
	r.Header.Set(PaymentHeader, "eyJ4IjoxfQ==")
	rec := httptest.NewRecorder()

	if cfg.Require(rec, r, "10000", "chat") {
		t.Fatal("Require should return false when settlement fails")
	}
	if rec.Code != http.StatusPaymentRequired {
		t.Errorf("status = %d, want 402", rec.Code)
	}
}

// TestPaymentSignatureHeaderAccepted checks the v2 request header is honored.
func TestPaymentSignatureHeaderAccepted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"isValid": true, "success": true, "transaction": "0x1"})
	}))
	defer srv.Close()

	cfg := Config{PayTo: "0xpay", FacilitatorURL: srv.URL}
	r := httptest.NewRequest(http.MethodGet, "/tool", nil)
	r.Header.Set(PaymentHeaderV2, "eyJ4IjoxfQ==") // PAYMENT-SIGNATURE only
	rec := httptest.NewRecorder()

	if !cfg.Require(rec, r, "10000", "chat") {
		t.Fatalf("Require should honor PAYMENT-SIGNATURE; body=%s", rec.Body.String())
	}
}

// TestRequirementsExtraForKnownNetwork checks the EIP-712 domain is filled in.
func TestRequirementsExtraForKnownNetwork(t *testing.T) {
	for _, net := range []string{"base", "eip155:8453"} {
		req := Config{PayTo: "0xpay", Network: net}.requirements("10000", "chat")
		if req.Extra["name"] == "" || req.Extra["version"] == "" {
			t.Errorf("network %q: extra not filled: %v", net, req.Extra)
		}
		if req.Asset == "" {
			t.Errorf("network %q: asset not defaulted", net)
		}
	}
}

// TestDecodePayment checks the base64 header is decoded to an object for the
// facilitator (which expects the payload object, not the raw string).
func TestDecodePayment(t *testing.T) {
	enc := base64.StdEncoding.EncodeToString([]byte(`{"network":"eip155:8453","payload":{"x":1}}`))
	obj := decodePayment(enc)
	m, ok := obj.(map[string]any)
	if !ok {
		t.Fatalf("decodePayment did not return an object: %T", obj)
	}
	if m["network"] != "eip155:8453" {
		t.Errorf("decoded network = %v", m["network"])
	}
	// Non-base64 passes through unchanged.
	if got := decodePayment("not-base64!"); got != "not-base64!" {
		t.Errorf("passthrough failed: %v", got)
	}
}

// TestCDPBearer certifies the CDP JWT is well-formed and its signature verifies.
func TestCDPBearer(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	secret := base64.StdEncoding.EncodeToString(priv)

	tok, err := cdpBearer("key-id", secret, "POST", "api.cdp.coinbase.com", "/platform/v2/x402/verify")
	if err != nil {
		t.Fatalf("cdpBearer: %v", err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("want 3 JWT segments, got %d", len(parts))
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	if !ed25519.Verify(pub, []byte(parts[0]+"."+parts[1]), sig) {
		t.Fatal("JWT signature does not verify")
	}
	var claims map[string]any
	cb, _ := base64.RawURLEncoding.DecodeString(parts[1])
	_ = json.Unmarshal(cb, &claims)
	if claims["iss"] != "cdp" || claims["uri"] != "POST api.cdp.coinbase.com/platform/v2/x402/verify" {
		t.Errorf("bad claims: %v", claims)
	}
}

// TestCDPAuthorizeAttachesBearer checks CDP() signs facilitator requests.
func TestCDPAuthorizeAttachesBearer(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	fac := CDP("key-id", base64.StdEncoding.EncodeToString(priv))
	req, _ := http.NewRequest(http.MethodPost, "https://api.cdp.coinbase.com/platform/v2/x402/verify", nil)
	if err := fac.Authorize(req); err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !strings.HasPrefix(req.Header.Get("Authorization"), "Bearer ey") {
		t.Errorf("missing Bearer JWT: %q", req.Header.Get("Authorization"))
	}
}

var _ Settler = (*HTTPFacilitator)(nil)
