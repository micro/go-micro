// Package provider is an external auth provider e.g oauth
package provider

import (
	"time"
)

// Provider is an auth provider
type Provider interface {
	// String returns the name of the provider
	String() string
	// Options returns the options of a provider
	Options() Options
	// Endpoint for the provider
	Endpoint() string
	// Redirect url incase of UI
	Redirect() string
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
