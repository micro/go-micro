// Package noop provides a no-op auth implementation for testing and development.
//
// The noop auth provider:
// - Accepts any token (always returns a valid account)
// - Grants all permissions (no actual authorization)
// - Generates tokens (but doesn't verify them)
//
// This is useful for:
// - Local development
// - Testing
// - Prototyping
//
// DO NOT use in production. Use JWT auth or implement a custom auth provider instead.
package noop

import (
	"go-micro.dev/v5/auth"
)

// NewAuth returns a new noop auth provider.
//
// The noop provider accepts all tokens and grants all permissions.
// This is for development and testing only - DO NOT use in production.
//
// Example:
//
//	authProvider := noop.NewAuth()
//	account, _ := authProvider.Generate("user123")
//	token, _ := authProvider.Token(auth.WithCredentials(account.ID, account.Secret))
func NewAuth(opts ...auth.Option) auth.Auth {
	return auth.NewAuth(opts...)
}

// NewRules returns a new noop rules implementation.
//
// The noop rules implementation grants all access and doesn't enforce any rules.
// This is for development and testing only.
//
// Example:
//
//	rules := noop.NewRules()
//	err := rules.Verify(account, resource) // Always returns nil
func NewRules() auth.Rules {
	return auth.NewRules()
}
