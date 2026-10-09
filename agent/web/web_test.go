package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchReturnsTextAndSourceLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><title>Example</title><script>do not include</script><p>Useful text</p><a href="/next">Next</a></html>`))
	}))
	defer server.Close()
	page, err := Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.Text, "Useful text") || !strings.Contains(page.Text, server.URL+"/next") || strings.Contains(page.Text, "do not include") {
		t.Fatalf("page=%+v", page)
	}
	if _, err := Fetch(context.Background(), "file:///etc/passwd"); err == nil {
		t.Fatal("accepted local file URL")
	}
}
