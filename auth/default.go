package auth

import (
	"github.com/google/uuid"
)

var (
	DefaultAuth = NewAuth()
)

func NewAuth(opts ...Option) Auth {
	options := Options{}

	for _, o := range opts {
		o(&options)
	}

	return &noop{
		opts: options,
	}
}

func NewRules() Rules {
	return new(noopRules)
}

type noop struct {
	opts Options
}

type noopRules struct{}

// String returns the name of the implementation
func (n *noop) String() string {
	return "noop"
}

// Init the auth
func (n *noop) Init(opts ...Option) {
	for _, o := range opts {
		o(&n.opts)
	}
}

// Options set for auth
func (n *noop) Options() Options {
	return n.opts
}

// Generate a new account
func (n *noop) Generate(id string, opts ...GenerateOption) (*Account, error) {
	options := NewGenerateOptions(opts...)

	return &Account{
		ID:       id,
		Secret:   options.Secret,
		Metadata: options.Metadata,
		Scopes:   options.Scopes,
		Issuer:   n.Options().Namespace,
	}, nil
}

// Grant access to a resource
func (n *noopRules) Grant(rule *Rule) error {
	return nil
}

// Revoke access to a resource
func (n *noopRules) Revoke(rule *Rule) error {
	return nil
}

// Rules used to verify requests
// Verify an account has access to a resource
func (n *noopRules) Verify(acc *Account, res *Resource, opts ...VerifyOption) error {
	return nil
}

func (n *noopRules) List(opts ...ListOption) ([]*Rule, error) {
	return []*Rule{}, nil
}

// Inspect a token
func (n *noop) Inspect(token string) (*Account, error) {
	return &Account{ID: uuid.New().String(), Issuer: n.Options().Namespace}, nil
}

// Token generation using an account id and secret
func (n *noop) Token(opts ...TokenOption) (*Token, error) {
	return &Token{}, nil
}
