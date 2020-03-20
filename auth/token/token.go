package token

import (
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a token cannot be found
	ErrNotFound = errors.New("token not found")
	// ErrEncodingToken is returned when the service encounters an error during encoding
	ErrEncodingToken = errors.New("error encoding the token")
	// ErrInvalidToken is returned when the token provided is not valid
	ErrInvalidToken = errors.New("invalid token provided")
)

// Provider generates and inspects tokens
type Provider interface {
	Generate(subject string, opts ...GenerateOption) (*Token, error)
	Inspect(token string) (*Token, error)
	String() string
}

// Token can be short or long lived
type Token struct {
	// The token itself
	Token string `json:"token"`
	// Type of token, e.g. JWT
	Type string `json:"type"`
	// Time of token creation
	Created time.Time `json:"created"`
	// Time of token expiry
	Expiry time.Time `json:"expiry"`
	// Subject of the token, e.g. the account ID
	Subject string `json:"subject"`
	// Roles granted to the token
	Roles []string `json:"roles"`
	// Metadata embedded in the token
	Metadata map[string]string `json:"metadata"`
}
