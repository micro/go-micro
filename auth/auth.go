// Package auth provides authentication and authorization capability
package auth

// Auth providers authentication and authorization
type Auth interface {
	// Generate a new authorization token
	Generate(u string) (*Token, error)
	// Revoke an authorization token
	Revoke(t *Token) error
	// Verify a token
	Verify(t *Token) error
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
