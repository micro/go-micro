package registry_test

import (
	"testing"

	"github.com/micro/go-micro/registry"
)

func TestOAuth2ClientCredentials(t *testing.T) {
	clientID := "client-id"
	clientSecret := "client-secret"
	tokenURL := "token-url"

	options := &registry.Options{}
	registry.OAuth2ClientCredentials(clientID, clientSecret, tokenURL)(options)

	creds := options.OAuth2ClientCredentials
	if creds == nil {
		t.Errorf("options.OAuth2ClientCredentials not set")
	}

	if clientID != creds.ClientID {
		t.Errorf("ClientID: want %q, got %q", clientID, creds.ClientID)
	}

	if clientSecret != creds.ClientSecret {
		t.Errorf("ClientSecret: want %q, got %q", clientSecret, creds.ClientSecret)
	}

	if tokenURL != creds.TokenURL {
		t.Errorf("TokenURL: want %q, got %q", tokenURL, creds.TokenURL)
	}
}
