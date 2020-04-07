package auth

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/micro/go-micro/v2/auth"
)

func TestNamespaceFromRequest(t *testing.T) {
	tt := []struct {
		Host      string
		Namespace string
	}{
		{Host: "micro.mu", Namespace: auth.DefaultNamespace},
		{Host: "micro.com.au", Namespace: auth.DefaultNamespace},
		{Host: "web.micro.mu", Namespace: auth.DefaultNamespace},
		{Host: "api.micro.mu", Namespace: auth.DefaultNamespace},
		{Host: "myapp.com", Namespace: auth.DefaultNamespace},
		{Host: "staging.myapp.com", Namespace: "staging"},
		{Host: "staging.myapp.m3o.app", Namespace: "myapp.staging"},
		{Host: "127.0.0.1", Namespace: auth.DefaultNamespace},
		{Host: "localhost", Namespace: auth.DefaultNamespace},
		{Host: "81.151.101.146", Namespace: auth.DefaultNamespace},
	}

	h := &authHandler{namespace: "domain"}

	for _, tc := range tt {
		t.Run(tc.Host, func(t *testing.T) {
			ns := h.NamespaceFromRequest(&http.Request{Host: tc.Host, URL: &url.URL{Host: tc.Host}})
			if ns != tc.Namespace {
				t.Errorf("Expected namespace %v for host %v, actually got %v", tc.Namespace, tc.Host, ns)
			}
		})
	}
}
