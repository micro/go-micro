package nano

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"go-micro.dev/v6/wrapper/x402"
)

type gatewayFixture struct {
	node   fakeNode
	rpc    string
	payTo  string
	payer  string
	price  string
	now    time.Time
	path   string
	claims *FileClaimStore
	fac    *Facilitator
}

func newGatewayFixture(t *testing.T) *gatewayFixture {
	t.Helper()
	payTo, _ := EncodeAccount(bytes.Repeat([]byte{0x22}, 32))
	payer, _ := EncodeAccount(bytes.Repeat([]byte{0x11}, 32))
	price, _ := RawFromXNO("0.001")
	node := fakeNode{}
	srv := httptest.NewServer(node)
	t.Cleanup(srv.Close)
	g := &gatewayFixture{
		node: node, rpc: srv.URL, payTo: payTo, payer: payer, price: price.String(),
		now: time.Unix(1_800_000_000, 0), path: filepath.Join(t.TempDir(), "claims"),
	}
	g.open(t)
	return g
}

// open starts a facilitator over the claims file, as a process start would.
func (g *gatewayFixture) open(t *testing.T) {
	t.Helper()
	claims, err := OpenFileClaimStore(g.path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { claims.Close() })
	fac, err := NewFacilitator(g.rpc, g.payTo, claims)
	if err != nil {
		t.Fatal(err)
	}
	now := g.now
	fac.Verifier().Now = func() time.Time { return now }
	g.claims, g.fac = claims, fac
}

func (g *gatewayFixture) handler() http.Handler {
	pay := x402.Middleware(x402.Config{
		PayTo:       g.payTo,
		Network:     "nano",
		Amount:      g.price,
		Facilitator: g.fac,
	})
	return pay(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("paid content"))
	}))
}

func (g *gatewayFixture) do(t *testing.T, payment string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/paid", nil)
	if payment != "" {
		r.Header.Set(x402.PaymentHeader, payment)
	}
	w := httptest.NewRecorder()
	g.handler().ServeHTTP(w, r)
	return w
}

func challengeOf(t *testing.T, w *httptest.ResponseRecorder) x402.Requirements {
	t.Helper()
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402; body %s", w.Code, w.Body.String())
	}
	var body struct {
		Accepts []x402.Requirements `json:"accepts"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || len(body.Accepts) != 1 {
		t.Fatalf("challenge body %s: %v", w.Body.String(), err)
	}
	return body.Accepts[0]
}

func paymentHeader(scheme, network, orderID, block string) string {
	raw, _ := json.Marshal(map[string]any{
		"x402Version": 1,
		"scheme":      scheme,
		"network":     network,
		"payload":     map[string]string{"orderId": orderID, "block": block},
	})
	return base64.StdEncoding.EncodeToString(raw)
}

func TestChallengeCarriesAFreshBoundOrder(t *testing.T) {
	g := newGatewayFixture(t)
	a := challengeOf(t, g.do(t, ""))
	b := challengeOf(t, g.do(t, ""))
	if a.Network != "nano:mainnet" || a.Scheme != "exact" || a.Asset != "XNO" || a.PayTo != g.payTo || a.MaxAmountRequired != g.price {
		t.Fatalf("challenge = %+v", a)
	}
	if !ValidOrderID(a.Extra["orderId"]) || a.Extra["orderId"] == b.Extra["orderId"] {
		t.Fatalf("order ids %q %q: want two fresh 128-bit ids", a.Extra["orderId"], b.Extra["orderId"])
	}
	if want := CommitmentAccount(a.Extra["orderId"], g.payTo, g.price); a.Extra["representative"] != want {
		t.Fatalf("representative = %q, want %q", a.Extra["representative"], want)
	}
}

func TestBoundPaymentServesOnceAndSurvivesRestart(t *testing.T) {
	g := newGatewayFixture(t)
	c := challengeOf(t, g.do(t, ""))
	order := c.Extra["orderId"]
	g.node[hashA] = sendBlock(g.payer, g.payTo, g.price, c.Extra["representative"], g.now.Add(-time.Minute))
	pay := paymentHeader("exact", "nano:mainnet", order, hashA)

	w := g.do(t, pay)
	if w.Code != http.StatusOK || w.Body.String() != "paid content" {
		t.Fatalf("paid request: %d %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get(x402.PaymentResponseHeader); got != hashA {
		t.Fatalf("settlement header = %q, want the block hash", got)
	}

	// The same block again is refused, and so is the same order with a
	// different block.
	challengeOf(t, g.do(t, pay))
	g.node[hashB] = sendBlock(g.payer, g.payTo, g.price, c.Extra["representative"], g.now.Add(-time.Minute))
	challengeOf(t, g.do(t, paymentHeader("exact", "nano:mainnet", order, hashB)))

	// A restart over the same claims file still refuses the block.
	g.claims.Close()
	g.open(t)
	challengeOf(t, g.do(t, pay))
}

func TestPaymentRefusals(t *testing.T) {
	g := newGatewayFixture(t)
	c := challengeOf(t, g.do(t, ""))
	order := c.Extra["orderId"]
	other, _ := NewOrderID()
	short := "999999999999999999999999999" // less than the price

	g.node[hashA] = sendBlock(g.payer, g.payTo, short, CommitmentAccount(order, g.payTo, short), g.now)
	g.node[hashB] = sendBlock(g.payer, g.payTo, g.price, CommitmentAccount(other, g.payTo, g.price), g.now)
	for name, pay := range map[string]string{
		"wrong amount":     paymentHeader("exact", "nano:mainnet", order, hashA),
		"bound to another": paymentHeader("exact", "nano:mainnet", order, hashB),
		"unknown block":    paymentHeader("exact", "nano:mainnet", order, "00"+hashA[2:]),
		"other scheme":     paymentHeader("upto", "nano:mainnet", other, hashB),
		"other network":    paymentHeader("exact", "eip155:8453", other, hashB),
		"not base64 json":  "not-a-payment",
		"missing order id": paymentHeader("exact", "nano:mainnet", "", hashB),
	} {
		t.Run(name, func(t *testing.T) { challengeOf(t, g.do(t, pay)) })
	}
	// None of the refused attempts used up the order bound in block B.
	w := g.do(t, paymentHeader("exact", "nano:mainnet", other, hashB))
	if w.Code != http.StatusOK {
		t.Fatalf("valid payment after refusals: %d %s", w.Code, w.Body.String())
	}
}

func TestSettleOnlyWhatWasVerified(t *testing.T) {
	g := newGatewayFixture(t)
	req := x402.Requirements{Scheme: "exact", Network: "nano:mainnet", PayTo: g.payTo, MaxAmountRequired: g.price}
	order, _ := NewOrderID()
	res, err := g.fac.Settle(context.Background(), paymentHeader("exact", "nano:mainnet", order, hashA), req)
	if err != nil || res.Valid {
		t.Fatalf("settle without verify = %+v, %v", res, err)
	}
}

func TestVerifyRefusesRequirementsItDoesNotOwn(t *testing.T) {
	g := newGatewayFixture(t)
	order, _ := NewOrderID()
	other, _ := EncodeAccount(bytes.Repeat([]byte{0x33}, 32))
	pay := paymentHeader("exact", "nano:mainnet", order, hashA)
	for _, req := range []x402.Requirements{
		{Scheme: "exact", Network: "base", PayTo: g.payTo, MaxAmountRequired: g.price},
		{Scheme: "exact", Network: "nano", PayTo: other, MaxAmountRequired: g.price},
		{Scheme: "exact", Network: "nano", PayTo: g.payTo, MaxAmountRequired: "0"},
		{Scheme: "exact", Network: "nano", PayTo: g.payTo, MaxAmountRequired: "1e30"},
	} {
		res, err := g.fac.Verify(context.Background(), pay, req)
		if err != nil || res.Valid {
			t.Fatalf("Verify(%+v) = %+v, %v", req, res, err)
		}
	}
}

func TestChallengeLeavesOtherNetworksAlone(t *testing.T) {
	g := newGatewayFixture(t)
	req := x402.Requirements{Scheme: "exact", Network: "base", PayTo: "0xabc", MaxAmountRequired: "10000"}
	got := g.fac.Challenge(req)
	if got.Network != "base" || got.Extra != nil || got.Asset != "" {
		t.Fatalf("Challenge changed a base requirement: %+v", got)
	}
	if x402.NormalizeNetwork("nano") != Network || x402.NormalizeNetwork("NANO:mainnet") != Network {
		t.Fatal("NormalizeNetwork does not map nano")
	}
}

func TestNewFacilitatorConfiguration(t *testing.T) {
	payTo, _ := EncodeAccount(bytes.Repeat([]byte{0x22}, 32))
	claims, err := OpenFileClaimStore(filepath.Join(t.TempDir(), "claims"))
	if err != nil {
		t.Fatal(err)
	}
	defer claims.Close()
	bad := payTo[:len(payTo)-1] + "1"
	if bad == payTo {
		bad = payTo[:len(payTo)-1] + "3"
	}
	for _, tc := range []struct {
		rpc, payTo string
		claims     ClaimStore
	}{
		{"https://node.example/rpc", bad, claims},
		{"http://node.example/rpc", payTo, claims},
		{"https://node.example/rpc", payTo, nil},
	} {
		if _, err := NewFacilitator(tc.rpc, tc.payTo, tc.claims); err == nil {
			t.Fatalf("NewFacilitator(%q, %q) accepted", tc.rpc, tc.payTo)
		}
	}
	if _, err := NewFacilitator("http://127.0.0.1:7076", payTo, claims); err != nil {
		t.Fatalf("loopback node refused: %v", err)
	}
}
