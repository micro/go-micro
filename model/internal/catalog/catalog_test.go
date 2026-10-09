package catalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestCatalogFormats(t *testing.T) {
	for _, format := range []string{"openai", "anthropic", "gemini", "together"} {
		t.Run(format, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch format {
				case "anthropic":
					if r.Header.Get("x-api-key") != "secret" {
						t.Error("missing key")
					}
					if calls == 1 {
						fmt.Fprint(w, `{"data":[{"id":"first"}],"has_more":true,"last_id":"cursor"}`)
					} else {
						if r.URL.Query().Get("after_id") != "cursor" {
							t.Error("missing cursor")
						}
						fmt.Fprint(w, `{"data":[{"id":"second"}]}`)
					}
				case "gemini":
					if r.Header.Get("x-goog-api-key") != "secret" {
						t.Error("missing key")
					}
					fmt.Fprint(w, `{"models":[{"name":"models/first","supportedGenerationMethods":["generateContent"]},{"name":"models/embedding","supportedGenerationMethods":["embedContent"]}]}`)
				case "together":
					fmt.Fprint(w, `[{"id":"first","type":"chat"},{"id":"image","type":"image"}]`)
				default:
					if r.Header.Get("Authorization") != "Bearer secret" {
						t.Error("missing key")
					}
					fmt.Fprint(w, `{"data":[{"id":"first"}]}`)
				}
			}))
			defer server.Close()
			got, err := List(context.Background(), server.URL, "secret", format)
			want := []string{"first"}
			if format == "anthropic" {
				want = append(want, "second")
			}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("%v %v", got, err)
			}
		})
	}
}

func TestCatalogDoesNotFollowRedirect(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer source.Close()
	if _, err := List(context.Background(), source.URL, "secret", "anthropic"); err == nil || called {
		t.Fatal("followed credential-bearing redirect")
	}
}
