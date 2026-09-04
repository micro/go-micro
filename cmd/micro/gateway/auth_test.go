package gateway

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/store"
	"golang.org/x/crypto/bcrypt"
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

func TestEnsureAdminFromEnv(t *testing.T) {
	setEnv := func(k, v string) {
		t.Helper()
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
	t.Cleanup(func() { setEnv("MICRO_ADMIN_PASSWORD", ""); setEnv("MICRO_ADMIN_USER", "") })

	st := store.NewMemoryStore()
	readAcc := func(t *testing.T, id string) Account {
		t.Helper()
		recs, _ := st.Read("auth/" + id)
		if len(recs) == 0 {
			t.Fatalf("no account %q in store", id)
		}
		var acc Account
		if err := json.Unmarshal(recs[0].Value, &acc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return acc
	}

	// No env → no account created.
	setEnv("MICRO_ADMIN_PASSWORD", "")
	if err := ensureAdminFromEnv(st); err != nil {
		t.Fatalf("ensureAdminFromEnv: %v", err)
	}
	if recs, _ := st.Read("auth/admin"); len(recs) != 0 {
		t.Fatal("account created without MICRO_ADMIN_PASSWORD")
	}

	// Env set → admin account created, password hashed and verifiable.
	setEnv("MICRO_ADMIN_PASSWORD", "micro")
	if err := ensureAdminFromEnv(st); err != nil {
		t.Fatalf("ensureAdminFromEnv: %v", err)
	}
	admin := readAcc(t, "admin")
	if admin.Type != "admin" || len(admin.Scopes) != 1 || admin.Scopes[0] != "*" {
		t.Fatalf("admin account wrong: %+v", admin)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Metadata["password_hash"]), []byte("micro")); err != nil {
		t.Fatal("stored password hash does not match 'micro'")
	}

	// Idempotent: existing account is not overwritten.
	admin.Metadata["changed"] = "true"
	b, _ := json.Marshal(admin)
	_ = st.Write(&store.Record{Key: "auth/admin", Value: b})
	if err := ensureAdminFromEnv(st); err != nil {
		t.Fatalf("ensureAdminFromEnv: %v", err)
	}
	if readAcc(t, "admin").Metadata["changed"] != "true" {
		t.Fatal("existing account was overwritten")
	}

	// A deleted admin stays deleted.
	_ = st.Delete("auth/admin")
	_ = st.Write(&store.Record{Key: "auth/.admin-deleted", Value: []byte("true")})
	if err := ensureAdminFromEnv(st); err != nil {
		t.Fatalf("ensureAdminFromEnv: %v", err)
	}
	if recs, _ := st.Read("auth/admin"); len(recs) != 0 {
		t.Fatal("deleted admin was recreated")
	}
}
