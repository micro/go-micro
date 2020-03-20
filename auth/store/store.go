package store

import (
	"encoding/json"
	"strings"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/auth/token"
	"github.com/micro/go-micro/v2/auth/token/basic"
	"github.com/micro/go-micro/v2/store"
	memStore "github.com/micro/go-micro/v2/store/memory"
)

// NewAuth returns a new default registry which is store
func NewAuth(opts ...auth.Option) auth.Auth {
	var s Store
	s.Init(opts...)
	return &s
}

// Store implementation of auth
type Store struct {
	secretProvider token.Provider
	tokenProvider  token.Provider
	opts           auth.Options
}

// String returns store
func (s *Store) String() string {
	return "store"
}

// Init the auth
func (s *Store) Init(opts ...auth.Option) {
	for _, o := range opts {
		o(&s.opts)
	}

	// use a memory store
	if s.opts.Store == nil {
		s.opts.Store = memStore.NewStore()
	}

	// use the memory store as the default
	// with a basic token provider, this is
	// pluggable to enable better testing
	if s.tokenProvider == nil {
		s.tokenProvider = basic.NewTokenProvider(token.WithStore(s.opts.Store))
	}
	if s.secretProvider == nil {
		s.secretProvider = basic.NewTokenProvider(token.WithStore(s.opts.Store))
	}
}

// Options returns the options
func (s *Store) Options() auth.Options {
	return s.opts
}

// Generate a new account
func (s *Store) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	// parse the options
	options := auth.NewGenerateOptions(opts...)

	// Generate a long-lived secret
	secretOpts := []token.GenerateOption{
		token.WithExpiry(options.SecretExpiry),
		token.WithMetadata(options.Metadata),
		token.WithRoles(options.Roles),
	}
	secret, err := s.secretProvider.Generate(id, secretOpts...)
	if err != nil {
		return nil, err
	}

	// return the account
	return &auth.Account{
		ID:       id,
		Roles:    options.Roles,
		Metadata: options.Metadata,
		Secret:   secret,
	}, nil
}

type rule struct {
	Role     string         `json:"rule"`
	Resource *auth.Resource `json:"resource"`
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
func (s *Store) Grant(role string, res *auth.Resource) error {
	r := rule{role, res}
	return s.opts.Store.Write(&store.Record{Key: r.Key(), Value: r.Bytes()})
}

// Revoke access to a resource
func (s *Store) Revoke(role string, res *auth.Resource) error {
	r := rule{role, res}

	err := s.opts.Store.Delete(r.Key())
	if err == store.ErrNotFound {
		return auth.ErrNotFound
	}

	return err
}

// Verify an account has access to a resource
func (s *Store) Verify(acc *auth.Account, res *auth.Resource) error {
	queries := [][]string{
		{res.Type, "*"},                         // check for wildcard resource type, e.g. service.*
		{res.Type, res.Name, "*"},               // check for wildcard name, e.g. service.foo*
		{res.Type, res.Name, res.Endpoint, "*"}, // check for wildcard endpoints, e.g. service.foo.ListFoo:*
		{res.Type, res.Name, res.Endpoint},      // check for specific role, e.g. service.foo.ListFoo:admin
	}

	for _, q := range queries {
		rules, err := s.listRules(q...)
		if err != nil {
			return err
		}

		for _, rule := range rules {
			if isValidRule(rule, acc, res) {
				return nil
			}
		}
	}

	return auth.ErrForbidden
}

func isValidRule(rule rule, acc *auth.Account, res *auth.Resource) bool {
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

func (s *Store) listRules(filters ...string) ([]rule, error) {
	// get the records from the store
	prefix := strings.Join(filters, "/")
	recs, err := s.opts.Store.Read(prefix, store.ReadPrefix())
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
func (s *Store) Inspect(t string) (*auth.Account, error) {
	tok, err := s.tokenProvider.Inspect(t)
	if err == token.ErrInvalidToken || err == token.ErrNotFound {
		return nil, auth.ErrInvalidToken
	} else if err != nil {
		return nil, err
	}

	return &auth.Account{
		ID:       tok.Subject,
		Roles:    tok.Roles,
		Metadata: tok.Metadata,
	}, nil
}

// Refresh an account using a secret
func (s *Store) Refresh(secret string) (*auth.Token, error) {
	sec, err := s.secretProvider.Inspect(secret)
	if err == token.ErrInvalidToken || err == token.ErrNotFound {
		return nil, auth.ErrInvalidToken
	} else if err != nil {
		return nil, err
	}

	opts := []token.GenerateOption{
		token.WithMetadata(sec.Metadata),
		token.WithRoles(sec.Roles),
	}

	return s.tokenProvider.Generate(sec.Subject, opts...)
}
