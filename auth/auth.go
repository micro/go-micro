// Package auth provides authentication and authorization capability
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/micro/go-micro/v2/metadata"
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
	// BearerScheme used for Authorization header
	BearerScheme = "Bearer "
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
	// Token generated using an account ID and secret
	Token(id, secret string, opts ...TokenOption) (*Token, error)
	// String returns the name of the implementation
	String() string
}

// Resource is an entity such as a user or
type Resource struct {
	// Name of the resource
	Name string
	// Type of resource, e.g.
	Type string
	// Endpoint resource e.g NotesService.Create
	Endpoint string
}

// Account provided by an auth provider
type Account struct {
	// ID of the account (UUIDV4, email or username)
	ID string `json:"id"`
	// Secret used to renew the account
	Secret string `json:"secret"`
	// Roles associated with the Account
	Roles []string `json:"roles"`
	// Any other associated metadata
	Metadata map[string]string `json:"metadata"`
	// Namespace the account belongs to, default blank
	Namespace string `json:"namespace"`
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
	// Namespace the token belongs to
	Namespace string `json:"namespace"`
}

const (
	// MetadataKey is the key used when storing the account in metadata
	MetadataKey = "auth-account"
	// TokenCookieName is the name of the cookie which stores the auth token
	TokenCookieName = "micro-token"
	// SecretCookieName is the name of the cookie which stores the auth secret
	SecretCookieName = "micro-secret"
)

// AccountFromContext gets the account from the context, which
// is set by the auth wrapper at the start of a call. If the account
// is not set, a nil account will be returned. The error is only returned
// when there was a problem retrieving an account
func AccountFromContext(ctx context.Context) (*Account, error) {
	str, ok := metadata.Get(ctx, MetadataKey)
	// there was no account set
	if !ok {
		return nil, nil
	}

	var acc *Account
	// metadata is stored as a string, so unmarshal to an account
	if err := json.Unmarshal([]byte(str), &acc); err != nil {
		return nil, err
	}

	return acc, nil
}

// ContextWithAccount sets the account in the context
func ContextWithAccount(ctx context.Context, account *Account) (context.Context, error) {
	// metadata is stored as a string, so marshal to bytes
	bytes, err := json.Marshal(account)
	if err != nil {
		return ctx, err
	}

	// generate a new context with the MetadataKey set
	return metadata.Set(ctx, MetadataKey, string(bytes)), nil
}

// ContextWithToken sets the auth token in the context
func ContextWithToken(ctx context.Context, token string) context.Context {
	return metadata.Set(ctx, "Authorization", fmt.Sprintf("%v%v", BearerScheme, token))
}
