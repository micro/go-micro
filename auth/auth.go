// Package auth provides authentication and authorization capability
package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/micro/go-micro/v2/metadata"
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
	// ID of the account (UUIDV4, email or username)
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

const (
	// MetadataKey is the key used when storing the account
	// in metadata
	MetadataKey = "auth-account"
	// CookieName is the name of the cookie which stores the
	// auth token
	CookieName = "micro-token"
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
