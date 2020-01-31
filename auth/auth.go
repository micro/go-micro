// Package auth provides authentication and authorization capability
package auth

import (
	"time"
)

// Auth providers authentication and authorization
type Auth interface {
	// Init the auth package
	Init(opts ...Option) error
	// Generate a new auth Account
	Generate(*Account) (*Account, error)
	// Revoke an authorization Account
	Revoke(string) error
	// Validate an account token
	Validate(string) (*Account, error)
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
	Name     string
	Resource *Resource
}

// Account provided by an auth provider
type Account struct {
	// The parent of the account, e.g. a user
	Parent *Resource `json: "parent"`
	// Token used to authenticate
	Token string `json: "token"`
	// Time of Account creation
	Created time.Time `json:"created"`
	// Time of Account expiry
	Expiry time.Time `json:"expiry"`
	// Roles associated with the Account
	Roles []*Role `json:"roles"`
	// Any other associated metadata
	Metadata map[string]string `json:"metadata"`
}
