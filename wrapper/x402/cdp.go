package x402

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CDPFacilitatorURL is Coinbase Developer Platform's hosted x402 facilitator,
// which can settle real payments on Base mainnet (the open x402.org facilitator
// is testnet-only).
const CDPFacilitatorURL = "https://api.cdp.coinbase.com/platform/v2/x402"

// CDP returns a Facilitator (and Settler) backed by Coinbase's hosted
// facilitator, authenticating each verify/settle call with a short-lived
// Ed25519 Bearer JWT minted from a CDP Secret API Key. keyID and keySecret are
// the CDP API Key ID and base64 Ed25519 secret; the secret is used only to sign
// the JWT (stdlib crypto — no chain code, no external dependency).
//
//	cfg.Facilitator = x402.CDP(os.Getenv("CDP_API_KEY_ID"), os.Getenv("CDP_API_KEY_SECRET"))
func CDP(keyID, keySecret string) *HTTPFacilitator {
	return &HTTPFacilitator{
		URL:       CDPFacilitatorURL,
		Authorize: cdpAuthorizer(keyID, keySecret),
	}
}

// cdpAuthorizer returns an Authorize hook that attaches a CDP Bearer JWT bound
// to the request's method and URL.
func cdpAuthorizer(keyID, keySecret string) func(*http.Request) error {
	return func(r *http.Request) error {
		tok, err := cdpBearer(keyID, keySecret, r.Method, r.URL.Host, r.URL.Path)
		if err != nil {
			return err
		}
		r.Header.Set("Authorization", "Bearer "+tok)
		return nil
	}
}

// cdpBearer builds a CDP Bearer JWT (EdDSA / Ed25519) authorizing a single REST
// call, per CDP's authentication spec: the token binds to "METHOD host/path"
// and is valid for two minutes.
func cdpBearer(keyID, keySecret, method, host, path string) (string, error) {
	if keyID == "" || keySecret == "" {
		return "", fmt.Errorf("CDP API key id and secret are required")
	}
	key, err := ed25519KeyFromSecret(keySecret)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	header := map[string]any{
		"alg":   "EdDSA",
		"typ":   "JWT",
		"kid":   keyID,
		"nonce": hex.EncodeToString(nonce),
	}
	now := time.Now().Unix()
	claims := map[string]any{
		"sub": keyID,
		"iss": "cdp",
		"aud": []string{"cdp_service"},
		"nbf": now,
		"exp": now + 120,
		"uri": method + " " + host + path,
	}
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	signing := b64url(hb) + "." + b64url(cb)
	sig := ed25519.Sign(key, []byte(signing))
	return signing + "." + b64url(sig), nil
}

// ed25519KeyFromSecret decodes a CDP Ed25519 secret. CDP secrets are base64 and
// decode to 64 bytes (32-byte seed + 32-byte public key) — Go's PrivateKey
// layout; a bare 32-byte seed is also accepted.
func ed25519KeyFromSecret(secret string) (ed25519.PrivateKey, error) {
	secret = strings.TrimSpace(secret)
	if strings.Contains(secret, "BEGIN") {
		return nil, fmt.Errorf("CDP secret looks like a PEM/EC key; x402 bearer auth needs an Ed25519 Secret API Key")
	}
	raw, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		if raw, err = base64.RawURLEncoding.DecodeString(secret); err != nil {
			return nil, fmt.Errorf("CDP secret is not valid base64: %w", err)
		}
	}
	switch len(raw) {
	case ed25519.PrivateKeySize: // 64: seed + public key
		return ed25519.PrivateKey(raw), nil
	case ed25519.SeedSize: // 32: seed only
		return ed25519.NewKeyFromSeed(raw), nil
	default:
		return nil, fmt.Errorf("CDP secret decoded to %d bytes; expected 32 or 64 (Ed25519)", len(raw))
	}
}

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
