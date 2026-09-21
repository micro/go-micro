package browserorigin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginValidation(t *testing.T) {
	for _, tt := range []struct {
		origin string
		want   bool
	}{
		{"http://localhost:1234", true}, {"https://localhost:1234", false}, {"http://localhost:9999", false}, {"http://localhost:1234.evil.example", false}, {"http://user@localhost:1234", false}, {"null", false}, {"http://localhost:1234/path", false}, {"http://localhost:1234#fragment", false}, {"https://trusted.example", true}, {"*", false},
	} {
		r := httptest.NewRequest(http.MethodPost, "http://localhost:1234/mcp", nil)
		r.Header.Set("Origin", tt.origin)
		if got := Allowed(r, []string{"https://trusted.example", "*"}); got != tt.want {
			t.Errorf("%q: %v", tt.origin, got)
		}
	}
	r := httptest.NewRequest(http.MethodPost, "http://localhost:1234/mcp", nil)
	if !Allowed(r, nil) {
		t.Fatal("native client rejected")
	}
	r.Header.Add("Origin", "http://localhost:1234")
	r.Header.Add("Origin", "https://trusted.example")
	if Allowed(r, []string{"https://trusted.example"}) {
		t.Fatal("multiple origins accepted")
	}
}
