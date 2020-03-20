package auth

import (
	"encoding/json"
	"strings"

	"github.com/micro/go-micro/v2/auth/token"
	"github.com/micro/go-micro/v2/auth/token/basic"
	"github.com/micro/go-micro/v2/store"
	memStore "github.com/micro/go-micro/v2/store/memory"
)

var (
	DefaultAuth = NewAuth()
)

// NewAuth returns a new default registry which is memory
func NewAuth(opts ...Option) Auth {
	var m memory
	m.Init(opts...)
	return &m
}

type memory struct {
	secretProvider token.Provider
	tokenProvider  token.Provider
	opts           Options
}

// String returns memory
func (m *memory) String() string {
	return "memory"
}

// Init the auth
func (m *memory) Init(opts ...Option) {
	for _, o := range opts {
		o(&m.opts)
	}

	// use a memory store
	if m.opts.Store == nil {
		m.opts.Store = memStore.NewStore()
	}

	// use the memory store as the default
	// with a basic token provider, this is
	// pluggable to enable better testing
	if m.tokenProvider == nil {
		m.tokenProvider = basic.NewTokenProvider(token.WithStore(m.opts.Store))
	}
	if m.secretProvider == nil {
		m.secretProvider = basic.NewTokenProvider(token.WithStore(m.opts.Store))
	}
}

// Options returns the options
func (m *memory) Options() Options {
	return m.opts
}

// Generate a new account
func (m *memory) Generate(id string, opts ...GenerateOption) (*Account, error) {
	// parse the options
	options := NewGenerateOptions(opts...)

	// Generate a long-lived secret
	secretOpts := []token.GenerateOption{
		token.WithExpiry(options.SecretExpiry),
		token.WithMetadata(options.Metadata),
		token.WithRoles(options.Roles),
	}
	secret, err := m.secretProvider.Generate(id, secretOpts...)
	if err != nil {
		return nil, err
	}

	// Generate the short-lived token
	tokenOpts := []token.GenerateOption{
		token.WithExpiry(options.TokenExpiry),
		token.WithMetadata(options.Metadata),
		token.WithRoles(options.Roles),
	}
	token, err := m.tokenProvider.Generate(id, tokenOpts...)
	if err != nil {
		return nil, err
	}

	// return the account
	return &Account{
		ID:       id,
		Roles:    options.Roles,
		Metadata: options.Metadata,
		Secret:   secret,
		Token:    token,
	}, nil
}

type rule struct {
	Role     string    `json:"rule"`
	Resource *Resource `json:"resource"`
}

func (r *rule) Key() string {
	comps := []string{r.Resource.Type, r.Resource.Name, r.Resource.Endpoint, r.Role}
	return strings.Join(comps, "/")
}

func (r *rule) Bytes() []byte {
	bytes, _ := json.Marshal(r)
	return bytes
}

// Grant access to a resource
func (m *memory) Grant(role string, res *Resource) error {
	r := rule{role, res}
	return m.opts.Store.Write(&store.Record{Key: r.Key(), Value: r.Bytes()})
}

// Revoke access to a resource
func (m *memory) Revoke(role string, res *Resource) error {
	r := rule{role, res}

	err := m.opts.Store.Delete(r.Key())
	if err == store.ErrNotFound {
		return ErrNotFound
	}

	return err
}

// Verify an account has access to a resource
func (m *memory) Verify(acc *Account, res *Resource) error {
	queries := [][]string{
		{res.Type, "*"},                         // check for wildcard resource type, e.g. service.*
		{res.Type, res.Name, "*"},               // check for wildcard name, e.g. service.foo*
		{res.Type, res.Name, res.Endpoint, "*"}, // check for wildcard endpoints, e.g. service.foo.ListFoo:*
		{res.Type, res.Name, res.Endpoint},      // check for specific role, e.g. service.foo.ListFoo:admin
	}

	for _, q := range queries {
		rules, err := m.listRules(q...)
		if err != nil {
			return err
		}

		for _, rule := range rules {
			if isValidRule(rule, acc, res) {
				return nil
			}
		}
	}

	return ErrForbidden
}

func isValidRule(rule rule, acc *Account, res *Resource) bool {
	if rule.Role == "*" {
		return true
	}

	for _, role := range acc.Roles {
		if rule.Role == role {
			return true
		}

		// allow user.anything if role is user.*
		if strings.HasSuffix(rule.Role, ".*") && strings.HasPrefix(rule.Role, role+".") {
			return true
		}
	}

	return false
}

func (m *memory) listRules(filters ...string) ([]rule, error) {
	// get the records from the store
	prefix := strings.Join(filters, "/")
	recs, err := m.opts.Store.Read(prefix, store.ReadPrefix())
	if err != nil {
		return nil, err
	}

	// unmarshal the records
	rules := make([]rule, 0, len(recs))
	for _, rec := range recs {
		var r rule
		if err := json.Unmarshal(rec.Value, &r); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	// return the rules
	return rules, nil
}

// Inspect a token
func (m *memory) Inspect(t string) (*Account, error) {
	tok, err := m.tokenProvider.Inspect(t)
	if err == token.ErrInvalidToken || err == token.ErrNotFound {
		return nil, ErrInvalidToken
	} else if err != nil {
		return nil, err
	}

	return &Account{
		ID:       tok.Subject,
		Roles:    tok.Roles,
		Metadata: tok.Metadata,
	}, nil
}

// Refresh an account using a secret
func (m *memory) Refresh(secret string) (*token.Token, error) {
	sec, err := m.secretProvider.Inspect(secret)
	if err == token.ErrInvalidToken || err == token.ErrNotFound {
		return nil, ErrInvalidToken
	} else if err != nil {
		return nil, err
	}

	opts := []token.GenerateOption{
		token.WithMetadata(sec.Metadata),
		token.WithRoles(sec.Roles),
	}

	return m.tokenProvider.Generate(sec.Subject, opts...)
}
