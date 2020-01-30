// Package auth provides authentication and authorization capability
package auth

import (
	"time"
)

// Auth providers authentication and authorization
type Auth interface {
	// Generate a new auth ServiceAccount
	Generate(string) (*ServiceAccount, error)
	// Revoke an authorization ServiceAccount
	Revoke(*ServiceAccount) error
	// AddRole to the service account
	AddRole(*ServiceAccount, *Role) error
	// RemoveRole from a service account
	RemoveRole(*ServiceAccount, *Role) error
}

// Resource is some thing to provide access to
type Resource struct {
	// Name of the resource
	Name string
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
	// Unique ServiceAccount id
	Id string `json: "id"`
	// Time of ServiceAccount creation
	Created time.Time `json:"created"`
	// Time of ServiceAccount expiry
	Expiry time.Time `json:"expiry"`
	// Roles associated with the ServiceAccount
	Roles []*Role `json:"roles"`
	// Any other associated metadata
	Metadata map[string]string `json:"metadata"`
}
