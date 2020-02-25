// Package auth provides authentication and authorization capability
package auth

import (
	"time"
)

// Auth providers authentication and authorization
type Auth interface {
	// Init the auth package
	Init(opts ...Option) error
	// Options returns the options set
	Options() Options
	// Generate a new auth Account
	Generate(id string, opts ...GenerateOption) (*Account, error)
	// Revoke an authorization Account
	Revoke(token string) error
	// Verify an account token
	Verify(token string) (*Account, error)
	// String returns the implementation
	String() string
}

// Resource is an entity such as a user or
type Resource struct {
	// Name of the resource
	Name string
	// Type of resource, e.g.
	Type string
}

// Role an account has
type Role struct {
	// Name of the role
	Name string
	// The resource it has access
	// TODO: potentially remove
	Resource *Resource
}

// Account provided by an auth provider
type Account struct {
	// ID of the account (UUID or email)
	Id string `json:"id"`
	// Token used to authenticate
	Token string `json:"token"`
	// Time of Account creation
	Created time.Time `json:"created"`
	// Time of Account expiry
	Expiry time.Time `json:"expiry"`
	// Roles associated with the Account
	Roles []*Role `json:"roles"`
	// Any other associated metadata
	Metadata map[string]string `json:"metadata"`
}
