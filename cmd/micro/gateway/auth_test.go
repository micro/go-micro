package gateway

import (
	"flag"
	"go-micro.dev/v6/store"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/urfave/cli/v2"
)

func TestIsExposed(t *testing.T) {
	cases := map[string]bool{
		":8080":            true,  // empty host = all interfaces
		"0.0.0.0:8080":     true,  // all interfaces
		"[::]:8080":        true,  // all interfaces (v6)
		"192.168.1.10:80":  true,  // routable
		"example.com:8080": true,  // hostname we can't classify → fail safe
		"127.0.0.1:8080":   false, // loopback
		"localhost:8080":   false, // loopback
		"[::1]:8080":       false, // loopback (v6)
	}
	for addr, want := range cases {
		if got := isExposed(addr); got != want {
			t.Errorf("isExposed(%q) = %v, want %v", addr, got, want)
		}
	}
}

func newCtx(t *testing.T, token string, auth, noAuth bool) *cli.Context {
	t.Helper()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.String("auth-token", token, "")
	set.Bool("auth", auth, "")
	set.Bool("no-auth", noAuth, "")
	return cli.NewContext(nil, set, nil)
}

func TestResolveAuthDefaultFollowsAddress(t *testing.T) {
	authToken = ""
	if enabled, _ := ResolveAuth(newCtx(t, "", false, false), "127.0.0.1:8080"); enabled {
		t.Fatal("loopback should default to auth off")
	}
	authToken = ""
	if enabled, _ := ResolveAuth(newCtx(t, "", false, false), ":8080"); !enabled {
		t.Fatal("exposed address should default to auth on")
	}
}

func TestResolveAuthOverrides(t *testing.T) {
	authToken = ""
	if enabled, _ := ResolveAuth(newCtx(t, "", false, true), ":8080"); enabled {
		t.Fatal("--no-auth must force auth off even when exposed")
	}
	authToken = ""
	if enabled, _ := ResolveAuth(newCtx(t, "", true, false), "127.0.0.1:8080"); !enabled {
		t.Fatal("--auth must force auth on even on loopback")
	}
}

func TestResolveAuthToken(t *testing.T) {
	// Supplied token is used verbatim and not echoed for printing.
	authToken = ""
	_, gen := ResolveAuth(newCtx(t, "supplied-secret", true, false), ":8080")
	if gen != "" {
		t.Fatalf("supplied token should not be returned for printing, got %q", gen)
	}
	if authToken != "supplied-secret" {
		t.Fatalf("authToken = %q, want the supplied secret", authToken)
	}
	if !tokenMatches("supplied-secret") || tokenMatches("wrong") {
		t.Fatal("tokenMatches should accept the supplied token and reject others")
	}

	// No token supplied → one is generated and returned to print once.
	authToken = ""
	_, gen = ResolveAuth(newCtx(t, "", true, false), ":8080")
	if gen == "" || gen != authToken {
		t.Fatalf("expected a generated token to be returned and stored, got gen=%q authToken=%q", gen, authToken)
	}
}

func TestTokenMatchesEmpty(t *testing.T) {
	authToken = ""
	if tokenMatches("") || tokenMatches("anything") {
		t.Fatal("an empty static token must never match")
	}
}

func TestMCPAuthUsesListenerAndGatewayToken(t *testing.T) {
	t.Setenv("MICRO_AUTH", "")
	old := authToken
	t.Cleanup(func() { authToken = old })
	for _, tc := range []struct {
		address          string
		force, off, want bool
	}{
		{"127.0.0.1:3000", false, false, false},
		{":3000", false, false, true},
		{"127.0.0.1:3000", true, false, true},
		{":3000", false, true, false},
	} {
		c := newCtx(t, "journey-token", tc.force, tc.off)
		ResolveAuth(c, "127.0.0.1:8080")
		opts, err := buildMCPOptions(c, tc.address)
		if err != nil {
			t.Fatal(err)
		}
		if (opts.Auth != nil) != tc.want {
			t.Fatalf("%+v: Auth = %v", tc, opts.Auth)
		}
		if !tc.want {
			continue
		}
		acc, err := opts.Auth.Inspect("journey-token")
		if err != nil || acc.ID != "admin" {
			t.Fatalf("machine token: %v %v", acc, err)
		}
		for _, token := range []string{"", "wrong", "journey-token"} {
			r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if token != "" {
				r.Header.Set("Authorization", "Bearer "+token)
			}
			err := opts.AuthFunc(r)
			if (err == nil) != (token == "journey-token") {
				t.Fatalf("token %q: %v", token, err)
			}
		}
	}
}

func TestMCPAuthJWTScopesAndRevocation(t *testing.T) {
	oldStore, oldPrivate, oldPublic := store.DefaultStore, jwtPrivateKey, jwtPublicKey
	store.DefaultStore = store.NewMemoryStore()
	t.Cleanup(func() { store.DefaultStore = oldStore; jwtPrivateKey = oldPrivate; jwtPublicKey = oldPublic })
	dir := t.TempDir()
	if err := InitJWTKeys(filepath.Join(dir, "private.pem"), filepath.Join(dir, "public.pem")); err != nil {
		t.Fatal(err)
	}
	token, err := GenerateJWT("reader", "user", []string{"notes:read"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	storeJWTToken(store.DefaultStore, token, "reader")
	a := gatewayAuth{}
	acc, err := a.Inspect(token)
	if err != nil || acc.ID != "reader" || len(acc.Scopes) != 1 || acc.Scopes[0] != "notes:read" {
		t.Fatalf("JWT: %v %v", acc, err)
	}
	if err := store.DefaultStore.Delete("jwt/" + token); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Inspect(token); err == nil {
		t.Fatal("revoked JWT accepted")
	}
}
