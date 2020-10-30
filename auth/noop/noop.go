package noop

import (
	"github.com/asim/nitro/v3/auth"
	"github.com/google/uuid"
)

func NewAuth(opts ...auth.Option) auth.Auth {
	var options auth.Options
	for _, o := range opts {
		o(&options)
	}

	return &noop{
		opts: options,
	}
}

func NewRules() auth.Rules {
	return &noopRules{}
}

type noop struct {
	opts auth.Options
}

// String returns the name of the implementation
func (n *noop) String() string {
	return "noop"
}

// Init the auth
func (n *noop) Init(opts ...auth.Option) {
	for _, o := range opts {
		o(&n.opts)
	}
}

// Options set for auth
func (n *noop) Options() auth.Options {
	return n.opts
}

// Generate a new account
func (n *noop) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	options := auth.NewGenerateOptions(opts...)
	name := options.Name
	if name == "" {
		name = id
	}
	return &auth.Account{
		ID:       id,
		Secret:   options.Secret,
		Metadata: options.Metadata,
		Scopes:   options.Scopes,
		Issuer:   n.Options().Issuer,
		Name:     name,
	}, nil
}

// Inspect a token
func (n *noop) Inspect(token string) (*auth.Account, error) {
	return &auth.Account{ID: uuid.New().String(), Issuer: n.Options().Issuer}, nil
}

// Token generation using an account id and secret
func (n *noop) Token(opts ...auth.TokenOption) (*auth.Token, error) {
	return &auth.Token{}, nil
}

type noopRules struct{}

// Grant access to a resource
func (n *noopRules) Grant(rule *auth.Rule) error {
	return nil
}

// Revoke access to a resource
func (n *noopRules) Revoke(rule *auth.Rule) error {
	return nil
}

func (n *noopRules) List(opts ...auth.RulesOption) ([]*auth.Rule, error) {
	return []*auth.Rule{}, nil
}

// Verify an account has access to a resource
func (n *noopRules) Verify(acc *auth.Account, res *auth.Resource, opts ...auth.VerifyOption) error {
	return nil
}
