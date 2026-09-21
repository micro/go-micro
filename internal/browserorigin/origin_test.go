package browserorigin

import (
	"context"
	"net"
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

func TestLoopbackSocketRejectsReboundBrowserHost(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://rebound.example:3000/mcp", nil)
	request = request.WithContext(context.WithValue(request.Context(), http.LocalAddrContextKey, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3000}))
	request.Header.Set("Origin", "http://rebound.example:3000")
	if Allowed(request, nil) {
		t.Fatal("browser-controlled Host authorized loopback access")
	}
	if !Allowed(request, []string{"http://rebound.example:3000"}) {
		t.Fatal("explicit proxy origin rejected")
	}
	request.Host = "localhost:3000"
	request.Header.Set("Origin", "http://localhost:3000")
	if !Allowed(request, nil) {
		t.Fatal("localhost origin rejected")
	}
}
