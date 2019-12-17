// Package auth provides authentication and authorization capability
package auth

import (
	"time"
)

// Auth providers authentication and authorization
type Auth interface {
	// Generate a new auth token
	Generate(string) (*Token, error)
	// Revoke an authorization token
	Revoke(*Token) error
	// Grant access to a resource
	Grant(*Token, *Service) error
	// Verify a token can access a resource
	Verify(*Token, *Service) error
}

// Service is some thing to provide access to
type Service struct {
	// Name of the resource
	Name string
	// Endpoint is the specific endpoint 
	Endpoint string
}

// Token providers by an auth provider
type Token struct {
	// Unique token id
	Id string `json: "id"`
	// Time of token creation
	Created time.Time `json:"created"`
	// Time of token expiry
	Expiry time.Time `json:"expiry"`
	// Roles associated with the token
	Roles []string `json:"roles"`
	// Any other associated metadata
	Metadata map[string]string `json:"metadata"`
}
