// Provider is an external auth provider e.g oauth
package provider

import (
	"context"
	"time"
)

// Provider is an external auth provider
type Provider interface {
	// Endpoint for the provider
	Endpoint() string
	// Redirect url incase of UI
	Redirect() string
	// Login to the provider using id and secret
	Login(ctx context.Context, id, secret string) (*Grant, error)
}

// Grant is a granted authorisation
type Grant struct {
	// token for reuse
	Token string
	// Expiry of the token
	Expiry time.Time
	// Scopes associated with grant
	Scopes []string
}
