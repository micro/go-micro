package nano

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Vectors from RFC 7693 and the xno-call-binding standard (BINDING-STANDARD.md).
const (
	burnAccount   = "nano_1111111111111111111111111111111111111111111111111111hifc8npp"
	vectorPrice   = "50000000000000000000000000000"
	vectorCommit  = "86e75a339b908cc926a478c1a0e5eb5dd6fc612c1d3251eb661faa78135c36cf"
	vectorRepAcct = "nano_33q9dassq66es6mcay83n5kypqgpzjikr9bkc9ope9xch1borfphgehj4ozj"
)

func TestBlake2bVectors(t *testing.T) {
	cases := []struct {
		in   []byte
		size int
		want string
	}{
		{[]byte("abc"), 64, "ba80a53f981c4d0d6a2797b69f12f6e94c212f14685ac4b74b12bb6fdbffa2d17d87c5392aab792dc252d5de4533cc9518d38aa8dbf1925ab92386edd4009923"},
		{nil, 5, "7d64c5272e"},
		{seq(200), 32, "63c3d97a9f8894d5e043a707b0fee7f7ec4c049a23bbf1079df20b4165f9e22d"},
	}
	for _, c := range cases {
		if got := hex.EncodeToString(blake2bSum(c.in, c.size)); got != c.want {
			t.Errorf("blake2b(%d bytes, %d) = %s, want %s", len(c.in), c.size, got, c.want)
		}
	}
}

func seq(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

func TestAccountEncodeDecode(t *testing.T) {
	got, err := EncodeAccount(make([]byte, 32))
	if err != nil || got != burnAccount {
		t.Fatalf("EncodeAccount(zero) = %q, %v", got, err)
	}
	pub := bytes.Repeat([]byte{0xab}, 32)
	acct, _ := EncodeAccount(pub)
	back, err := DecodeAccount(acct)
	if err != nil || !bytes.Equal(back, pub) {
		t.Fatalf("round trip failed: %v", err)
	}
	if _, err := DecodeAccount("xrb_" + acct[5:]); err != nil {
		t.Fatalf("legacy prefix refused: %v", err)
	}
}

func TestAccountRejectsBadChecksumAndShape(t *testing.T) {
	bad := []string{
		burnAccount[:len(burnAccount)-1] + "q",    // checksum changed
		"nano_3" + burnAccount[6:],                // key changed, checksum kept
		"nano_" + strings.Repeat("1", 59),         // too short
		"btc_" + burnAccount[5:],                  // wrong prefix
		strings.Replace(burnAccount, "1", "0", 3), // 0 is not in the alphabet
		"nano_4" + burnAccount[6:],                // pad bits set
	}
	for _, a := range bad {
		if ValidateAccount(a) == nil {
			t.Errorf("ValidateAccount(%q) accepted", a)
		}
	}
}

func TestRawFromXNO(t *testing.T) {
	good := map[string]string{
		"1":                                "1000000000000000000000000000000",
		"0.05":                             vectorPrice,
		"0.001":                            "1000000000000000000000000000",
		".5":                               "500000000000000000000000000000",
		"0.000000000000000000000000000001": "1",
	}
	for in, want := range good {
		got, err := RawFromXNO(in)
		if err != nil || got.String() != want {
			t.Errorf("RawFromXNO(%q) = %v, %v; want %s", in, got, err, want)
		}
	}
	for _, in := range []string{"", "0", "0.0", "-1", "1e3", "1.", "abc", "1,5", "0.0000000000000000000000000000001"} {
		if _, err := RawFromXNO(in); err == nil {
			t.Errorf("RawFromXNO(%q) accepted", in)
		}
	}
}

func TestCommitmentVectors(t *testing.T) {
	c := Commitment("order-0001", burnAccount, vectorPrice)
	if hex.EncodeToString(c[:]) != vectorCommit {
		t.Fatalf("commitment = %x", c)
	}
	if got := CommitmentAccount("order-0001", burnAccount, vectorPrice); got != vectorRepAcct {
		t.Fatalf("commitment account = %s", got)
	}
}

func TestNewOrderID(t *testing.T) {
	a, err := NewOrderID()
	if err != nil || !ValidOrderID(a) {
		t.Fatalf("NewOrderID = %q, %v", a, err)
	}
	b, _ := NewOrderID()
	if a == b {
		t.Fatal("two order ids are equal")
	}
}

const (
	hashA = "A1B2C3D4E5F60718293A4B5C6D7E8F90A1B2C3D4E5F60718293A4B5C6D7E8F90"
	hashB = "0F1E2D3C4B5A69788796A5B4C3D2E1F00F1E2D3C4B5A69788796A5B4C3D2E1F0"
)

func TestFileClaimStoreSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claims")
	order1, _ := NewOrderID()
	order2, _ := NewOrderID()

	s, err := OpenFileClaimStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(strings.ToLower(hashA), order1); err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(hashA, order2); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("same block, other order: %v", err)
	}
	if err := s.Claim(hashB, order1); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("same order, other block: %v", err)
	}
	s.Close()

	s2, err := OpenFileClaimStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if err := s2.Claim(hashA, order2); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("block replayed after restart: %v", err)
	}
	if err := s2.Claim(hashB, order2); err != nil {
		t.Fatalf("fresh claim after restart: %v", err)
	}
}

func TestFileClaimStoreRefusesMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claims")
	if err := os.WriteFile(path, []byte("not a claim\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenFileClaimStore(path); err == nil {
		t.Fatal("malformed claims file accepted")
	}

	// A bad record in the middle is corruption, even when a good one follows.
	order, _ := NewOrderID()
	middle := hashA + " " + order[:10] + "\n" + hashB + " " + order + "\n"
	if err := os.WriteFile(path, []byte(middle), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenFileClaimStore(path); err == nil {
		t.Fatal("claims file corrupt in the middle accepted")
	}
}

func TestFileClaimStoreDropsIncompleteLastRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claims")
	order1, _ := NewOrderID()
	order2, _ := NewOrderID()
	// A complete claim, then an append cut off by a crash or a short write.
	torn := hashA + " " + order1 + "\n" + hashB + " " + order2[:7]
	if err := os.WriteFile(path, []byte(torn), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := OpenFileClaimStore(path)
	if err != nil {
		t.Fatalf("torn tail refused: %v", err)
	}
	if err := s.Claim(hashA, order2); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("complete claim lost: %v", err)
	}
	// The torn claim was never reported as recorded, so it is claimable, and
	// its record must start on its own line.
	if err := s.Claim(hashB, order2); err != nil {
		t.Fatalf("claim after torn tail: %v", err)
	}
	s.Close()

	want := hashA + " " + order1 + "\n" + hashB + " " + order2 + "\n"
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("claims file = %q, want %q", got, want)
	}
	s2, err := OpenFileClaimStore(path)
	if err != nil {
		t.Fatalf("reopen after repair: %v", err)
	}
	s2.Close()
}

func TestFileClaimStoreIsSingleInstance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claims")
	s, err := OpenFileClaimStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if s2, err := OpenFileClaimStore(path); !errors.Is(err, ErrClaimsLocked) {
		if s2 != nil {
			s2.Close()
		}
		t.Fatalf("second store on the same file: %v", err)
	}
	s.Close()
	s3, err := OpenFileClaimStore(path)
	if err != nil {
		t.Fatalf("reopen after close: %v", err)
	}
	s3.Close()
}

func TestFileClaimStoreFsyncFailureKeepsClaim(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claims")
	order1, _ := NewOrderID()
	order2, _ := NewOrderID()
	s, err := OpenFileClaimStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.sync = func() error { return errors.New("injected fsync failure") }
	if err := s.Claim(hashA, order1); err == nil {
		t.Fatal("fsync failure not reported")
	}
	// The record may be on disk, so the block and order stay spent.
	if err := s.Claim(hashA, order2); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("block reusable after failed fsync: %v", err)
	}
	if err := s.Claim(hashB, order1); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("order reusable after failed fsync: %v", err)
	}
	// And the store fails closed until it is reopened.
	if err := s.Claim(hashB, order2); err == nil {
		t.Fatal("claim accepted after failed fsync")
	}
}

// fakeNode answers block_info for the blocks it holds.
type fakeNode map[string]map[string]any

func (n fakeNode) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	_ = json.NewDecoder(r.Body).Decode(&req)
	b, ok := n[req["hash"]]
	if req["action"] != "block_info" || !ok {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Block not found"})
		return
	}
	_ = json.NewEncoder(w).Encode(b)
}

func sendBlock(payer, payTo, amount, rep string, ts time.Time) map[string]any {
	return map[string]any{
		"block_account":   payer,
		"amount":          amount,
		"confirmed":       "true",
		"subtype":         "send",
		"local_timestamp": strconv.FormatInt(ts.Unix(), 10),
		"contents": map[string]string{
			"type":            "state",
			"representative":  rep,
			"link_as_account": payTo,
		},
	}
}

type verifyFixture struct {
	v     *Verifier
	node  fakeNode
	payTo string
	payer string
	price string
	now   time.Time
	path  string
}

func newFixture(t *testing.T) *verifyFixture {
	t.Helper()
	payTo, _ := EncodeAccount(bytes.Repeat([]byte{0x22}, 32))
	payer, _ := EncodeAccount(bytes.Repeat([]byte{0x11}, 32))
	price, _ := RawFromXNO("0.001")
	node := fakeNode{}
	srv := httptest.NewServer(node)
	t.Cleanup(srv.Close)
	path := filepath.Join(t.TempDir(), "claims")
	claims, err := OpenFileClaimStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { claims.Close() })
	v, err := NewVerifier(srv.URL, payTo, price, claims)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	v.Now = func() time.Time { return now }
	return &verifyFixture{v: v, node: node, payTo: payTo, payer: payer, price: price.String(), now: now, path: path}
}

func TestVerifyAcceptsBoundExactPaymentOnce(t *testing.T) {
	f := newFixture(t)
	order, _ := NewOrderID()
	offer := f.v.Offer(order)
	if offer.Network != "nano:mainnet" || offer.Scheme != "exact" || offer.Amount != f.price || offer.PayTo != f.payTo {
		t.Fatalf("offer = %+v", offer)
	}
	f.node[hashA] = sendBlock(f.payer, f.payTo, f.price, offer.Representative, f.now.Add(-time.Minute))

	p, err := f.v.Verify(context.Background(), order, strings.ToLower(hashA))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if p.Payer != f.payer || p.AmountRaw != f.price || p.BlockHash != hashA || p.OrderID != order {
		t.Fatalf("payment = %+v", p)
	}
	if _, err := f.v.Verify(context.Background(), order, hashA); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("replay: %v", err)
	}

	// A restart: a new verifier over the same claims file still refuses it.
	// The old store must be closed first; it holds the file's lock.
	f.v.Claims.(*FileClaimStore).Close()
	claims, err := OpenFileClaimStore(f.path)
	if err != nil {
		t.Fatal(err)
	}
	defer claims.Close()
	f.v.Claims = claims
	if _, err := f.v.Verify(context.Background(), order, hashA); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("replay after restart: %v", err)
	}
}

func TestVerifyRefusals(t *testing.T) {
	f := newFixture(t)
	order, _ := NewOrderID()
	other, _ := NewOrderID()
	rep := f.v.Offer(order).Representative
	recent := f.now.Add(-time.Minute)
	otherPayTo, _ := EncodeAccount(bytes.Repeat([]byte{0x33}, 32))
	over := new(big.Int).Add(f.v.PriceRaw, big.NewInt(1)).String()
	short := new(big.Int).Sub(f.v.PriceRaw, big.NewInt(1)).String()

	unconfirmed := sendBlock(f.payer, f.payTo, f.price, rep, recent)
	unconfirmed["confirmed"] = "false"
	receive := sendBlock(f.payer, f.payTo, f.price, rep, recent)
	receive["subtype"] = "receive"

	cases := []struct {
		name  string
		block map[string]any
		order string
		want  error
	}{
		{"wrong destination", sendBlock(f.payer, otherPayTo, f.price, rep, recent), order, ErrWrongPayTo},
		{"overpaid", sendBlock(f.payer, f.payTo, over, rep, recent), order, ErrWrongAmount},
		{"underpaid", sendBlock(f.payer, f.payTo, short, rep, recent), order, ErrWrongAmount},
		{"block for another order", sendBlock(f.payer, f.payTo, f.price, rep, recent), other, ErrNotBound},
		{"plain representative", sendBlock(f.payer, f.payTo, f.price, f.payer, recent), order, ErrNotBound},
		{"unconfirmed", unconfirmed, order, ErrNotConfirmed},
		{"not a send", receive, order, ErrNotConfirmed},
		{"too old", sendBlock(f.payer, f.payTo, f.price, rep, f.now.Add(-time.Hour)), order, ErrTooOld},
	}
	for _, c := range cases {
		f.node[hashA] = c.block
		if _, err := f.v.Verify(context.Background(), c.order, hashA); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
	if _, err := f.v.Verify(context.Background(), order, hashB); !errors.Is(err, ErrNotConfirmed) {
		t.Errorf("unknown block: %v", err)
	}
	if _, err := f.v.Verify(context.Background(), "not-an-order", hashA); !errors.Is(err, ErrInvalidOrderID) {
		t.Errorf("bad order id: %v", err)
	}
	if _, err := f.v.Verify(context.Background(), order, "xyz"); !errors.Is(err, ErrInvalidBlockID) {
		t.Errorf("bad block hash: %v", err)
	}

	// None of the refusals consumed the order: the right block still pays it.
	f.node[hashA] = sendBlock(f.payer, f.payTo, f.price, rep, recent)
	if _, err := f.v.Verify(context.Background(), order, hashA); err != nil {
		t.Fatalf("valid payment after refusals: %v", err)
	}
}

func TestNewVerifierConfiguration(t *testing.T) {
	payTo, _ := EncodeAccount(bytes.Repeat([]byte{0x22}, 32))
	price := big.NewInt(1)
	claims := &FileClaimStore{}
	if _, err := NewVerifier("http://node.example:7076", payTo, price, claims); err == nil {
		t.Error("plain http to a remote node accepted")
	}
	for _, u := range []string{"https://node.example/rpc", "http://127.0.0.1:7076", "http://localhost:7076"} {
		if _, err := NewVerifier(u, payTo, price, claims); err != nil {
			t.Errorf("%s refused: %v", u, err)
		}
	}
	if _, err := NewVerifier("https://node.example", payTo[:len(payTo)-1]+"1", price, claims); err == nil {
		t.Error("bad pay-to checksum accepted")
	}
	if _, err := NewVerifier("https://node.example", payTo, big.NewInt(0), claims); err == nil {
		t.Error("zero price accepted")
	}
	if _, err := NewVerifier("https://node.example", payTo, price, nil); err == nil {
		t.Error("missing claim store accepted")
	}
}
