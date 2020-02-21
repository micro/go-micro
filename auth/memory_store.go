package auth

import (
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/errors"
)

// newRegistry returns an instance of memory auth
func newRegistry(opts ...Option) Auth {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	return &memory{
		store: map[string]*Account{},
		opts:  options,
	}
}

type memory struct {
	store map[string]*Account
	opts  Options
}

// Init the auth package
func (m *memory) Init(opts ...Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}

// Options returns the options set
func (m *memory) Options() Options {
	return m.opts
}

// Generate a new auth Account
func (m *memory) Generate(id string, opts ...GenerateOption) (*Account, error) {
	// generate the token
	token, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	// parse the options
	options := NewGenerateOptions(opts...)

	// construct the account
	sa := &Account{
		Id:       id,
		Token:    token.String(),
		Created:  time.Now(),
		Metadata: options.Metadata,
		Roles:    options.Roles,
	}

	// store in memory
	m.store[sa.Token] = sa

	// return the result
	return sa, nil
}

// Revoke an authorization Account
func (m *memory) Revoke(token string) error {
	if _, ok := m.store[token]; !ok {
		return errors.BadRequest("go.micro.auth", "token not found")
	}
	delete(m.store, token)

	return nil
}

// Validate an account token
func (m *memory) Validate(token string) (*Account, error) {
	// lookup the record by token
	record, ok := m.store[token]
	if !ok {
		return nil, errors.Unauthorized("go.micro.auth", "invalid token")
	}

	// return the result
	return record, nil
}

// String returns the implementation
func (m *memory) String() string {
	return "memory"
}
