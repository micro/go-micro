package gateway

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/store"
)

func TestDashboardAdminBoundary(t *testing.T) {
	oldHTML, oldToken, oldPrivate, oldPublic := HTML, authToken, jwtPrivateKey, jwtPublicKey
	t.Cleanup(func() { HTML, authToken, jwtPrivateKey, jwtPublicKey = oldHTML, oldToken, oldPrivate, oldPublic })
	HTML = os.DirFS("..")
	authToken = "test-machine-token"
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwtPrivateKey, jwtPublicKey = key, &key.PublicKey
	for _, role := range []string{"user", "service", "admin"} {
		t.Run(role, func(t *testing.T) {
			st := store.NewMemoryStore()
			data, _ := json.Marshal(Account{ID: "caller", Type: role})
			if err := st.Write(&store.Record{Key: "auth/caller", Value: data}); err != nil {
				t.Fatal(err)
			}
			token, err := GenerateJWT("caller", role, []string{"*"}, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			storeJWTToken(st, token, "caller")
			mux := http.NewServeMux()
			registerHandlers(mux, parseTemplates(), st, true)
			for _, path := range []string{"/auth/users", "/auth/tokens", "/auth/scopes", "/auth/scopes/bulk"} {
				if role == "admin" && strings.Contains(path, "scopes") {
					continue
				} // discovery is irrelevant to this authorization regression
				request := httptest.NewRequest(http.MethodPost, path, strings.NewReader("id=created&type=admin&password=test"))
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				request.Header.Set("Authorization", "Bearer "+token)
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				want := http.StatusForbidden
				if role == "admin" {
					want = http.StatusSeeOther
				}
				if response.Code != want {
					t.Fatalf("%s: got %d want %d", path, response.Code, want)
				}
			}
			records, _ := st.Read("auth/created")
			if role != "admin" && len(records) > 0 {
				t.Fatal("non-admin modified accounts")
			}
			if role == "admin" {
				data, _ := json.Marshal(Account{ID: "caller", Type: "user"})
				_ = st.Write(&store.Record{Key: "auth/caller", Value: data})
				rr := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/auth/tokens", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				mux.ServeHTTP(rr, req)
				if rr.Code != http.StatusForbidden {
					t.Fatal("demoted admin retained access")
				}
			}
		})
	}
}

func TestDashboardTemplatesEscapeUntrustedText(t *testing.T) {
	old := HTML
	HTML = os.DirFS("..")
	t.Cleanup(func() { HTML = old })
	tmpls := parseTemplates()
	payload := `<script>untrusted()</script>`
	for _, tmpl := range []*template.Template{tmpls.api, tmpls.service, tmpls.form, tmpls.home, tmpls.logs, tmpls.log, tmpls.status, tmpls.authTokens, tmpls.authLogin, tmpls.authUsers, tmpls.playground, tmpls.scopes} {
		var out bytes.Buffer
		if err := tmpl.Execute(&out, map[string]any{"Title": payload, "User": &TemplateUser{ID: payload}, "Log": payload, "Error": payload, "ServiceName": payload}); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), payload) {
			t.Fatal("unescaped template value")
		}
		if !strings.Contains(out.String(), "&lt;script&gt;") {
			t.Fatal("missing escaped test value")
		}
	}
}
