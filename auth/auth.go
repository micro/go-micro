// Package auth provides authentication and authorization capability
package auth

import (
	"time"
)

// Auth providers authentication and authorization
type Auth interface {
	// Init the auth package
	Init(opts ...Option) error
	// Generate a new auth ServiceAccount
	Generate(*ServiceAccount) (*ServiceAccount, error)
	// Revoke an authorization ServiceAccount
	Revoke(string) error
	// Validate a service account token
	Validate(string) (*ServiceAccount, error)
}

// Resource is an entity such as a user or service
type Resource struct {
	// Id of the resource
	Id string
	// Type of resource, e.g. Service
	Type string
}

// Role a service account has
type Role struct {
	Name     string
	Resource *Resource
}

// ServiceAccount providers by an auth provider
type ServiceAccount struct {
	// The parent of the service account, e.g. a user
	Parent *Resource `json: "parent"`
	// Token used to authenticate
	Token string `json: "token"`
	// Time of ServiceAccount creation
	Created time.Time `json:"created"`
	// Time of ServiceAccount expiry
	Expiry time.Time `json:"expiry"`
	// Roles associated with the ServiceAccount
	Roles []*Role `json:"roles"`
	// Any other associated metadata
	Metadata map[string]string `json:"metadata"`
}
