// Package auth provides authentication and authorization capability
package auth

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a resouce cannot be found
	ErrNotFound = errors.New("not found")
	// ErrEncodingToken is returned when the service encounters an error during encoding
	ErrEncodingToken = errors.New("error encoding the token")
	// ErrInvalidToken is returned when the token provided is not valid
	ErrInvalidToken = errors.New("invalid token provided")
	// ErrInvalidRole is returned when the role provided was invalid
	ErrInvalidRole = errors.New("invalid role")
	// ErrForbidden is returned when a user does not have the necessary roles to access a resource
	ErrForbidden = errors.New("resource forbidden")
)

// Auth providers authentication and authorization
type Auth interface {
	// Init the auth
	Init(opts ...Option)
	// Options set for auth
	Options() Options
	// Generate a new account
	Generate(id string, opts ...GenerateOption) (*Account, error)
	// Grant access to a resource
	Grant(role string, res *Resource) error
	// Revoke access to a resource
	Revoke(role string, res *Resource) error
	// Verify an account has access to a resource
	Verify(acc *Account, res *Resource) error
	// Inspect a token
	Inspect(token string) (*Account, error)
	// Token generated using refresh token
	Token(opts ...TokenOption) (*Token, error)
	// String returns the name of the implementation
	String() string
}

// Resource is an entity such as a user or
type Resource struct {
	// Name of the resource
	Name string `json:"name"`
	// Type of resource, e.g.
	Type string `json:"type"`
	// Endpoint resource e.g NotesService.Create
	Endpoint string `json:"endpoint"`
	// Namespace the resource belongs to
	Namespace string `json:"namespace"`
}

// Account provided by an auth provider
type Account struct {
	// ID of the account e.g. email
	ID string `json:"id"`
	// Type of the account, e.g. service
	Type string `json:"type"`
	// Provider who issued the account
	Provider string `json:"provider"`
	// Roles associated with the Account
	Roles []string `json:"roles"`
	// Any other associated metadata
	Metadata map[string]string `json:"metadata"`
	// Namespace the account belongs to
	Namespace string `json:"namespace"`
	// Secret for the account, e.g. the password
	Secret string `json:"secret"`
}

// Token can be short or long lived
type Token struct {
	// The token to be used for accessing resources
	AccessToken string `json:"access_token"`
	// RefreshToken to be used to generate a new token
	RefreshToken string `json:"refresh_token"`
	// Time of token creation
	Created time.Time `json:"created"`
	// Time of token expiry
	Expiry time.Time `json:"expiry"`
}

const (
	// DefaultNamespace used for auth
	DefaultNamespace = "go.micro"
	// TokenCookieName is the name of the cookie which stores the auth token
	TokenCookieName = "micro-token"
	// BearerScheme used for Authorization header
	BearerScheme = "Bearer "
)

type accountKey struct{}

// AccountFromContext gets the account from the context, which
// is set by the auth wrapper at the start of a call. If the account
// is not set, a nil account will be returned. The error is only returned
// when there was a problem retrieving an account
func AccountFromContext(ctx context.Context) (*Account, bool) {
	acc, ok := ctx.Value(accountKey{}).(*Account)
	return acc, ok
}

// ContextWithAccount sets the account in the context
func ContextWithAccount(ctx context.Context, account *Account) context.Context {
	return context.WithValue(ctx, accountKey{}, account)
}
