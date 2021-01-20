package eureka

import (
	"context"
	"net/http"
	"testing"

	"golang.org/x/oauth2/clientcredentials"

	"github.com/micro/go-micro/v2/registry"
)

func TestOAuth2ClientCredentials(t *testing.T) {
	clientID := "client-id"
	clientSecret := "client-secret"
	tokenURL := "token-url"

	var config clientcredentials.Config

	origFn := newOAuthClient
	newOAuthClient = func(c clientcredentials.Config) *http.Client {
		config = c
		return origFn(c)
	}

	options := new(registry.Options)
	options.Context = context.WithValue(context.Background(), "foo", "bar")

	OAuth2ClientCredentials(clientID, clientSecret, tokenURL)(options)

	if clientID != config.ClientID {
		t.Errorf("ClientID: want %q, got %q", clientID, config.ClientID)
	}

	if clientSecret != config.ClientSecret {
		t.Errorf("ClientSecret: want %q, got %q", clientSecret, config.ClientSecret)
	}

	if tokenURL != config.TokenURL {
		t.Errorf("TokenURL: want %q, got %q", tokenURL, config.TokenURL)
	}

	if _, ok := options.Context.Value(contextHttpClient{}).(*http.Client); !ok {
		t.Errorf("HttpClient not set in options.Context")
	}

	if str, ok := options.Context.Value("foo").(string); !ok || str != "bar" {
		t.Errorf("Original context overwritten")
	}
}
