package basic

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/auth/token"
	"github.com/micro/go-micro/v2/store"
)

// Basic implementation of token provider, backed by the store
type Basic struct {
	store store.Store
}

// NewTokenProvider returns an initialized basic provider
func NewTokenProvider(opts ...token.Option) token.Provider {
	options := token.NewOptions(opts...)

	if options.Store == nil {
		options.Store = store.DefaultStore
	}

	return &Basic{
		store: options.Store,
	}
}

// Generate a token for an account
func (b *Basic) Generate(subject string, opts ...token.GenerateOption) (*auth.Token, error) {
	options := token.NewGenerateOptions(opts...)

	// construct the token
	token := auth.Token{
		Subject:  subject,
		Type:     b.String(),
		Token:    uuid.New().String(),
		Created:  time.Now(),
		Expiry:   time.Now().Add(options.Expiry),
		Metadata: options.Metadata,
		Roles:    options.Roles,
	}

	// marshal the account to bytes
	bytes, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}

	// write to the store
	err = b.store.Write(&store.Record{
		Key:    token.Token,
		Value:  bytes,
		Expiry: options.Expiry,
	})
	if err != nil {
		return nil, err
	}

	// return the token
	return &token, nil
}

// Inspect a token
func (b *Basic) Inspect(t string) (*auth.Token, error) {
	// lookup the token in the store
	recs, err := b.store.Read(t)
	if err == store.ErrNotFound {
		return nil, token.ErrInvalidToken
	} else if err != nil {
		return nil, err
	}
	bytes := recs[0].Value

	// unmarshal the bytes
	var tok *auth.Token
	if err := json.Unmarshal(bytes, &tok); err != nil {
		return nil, err
	}

	// ensure the token hasn't expired, the store should
	// expire the token but we're checking again
	if tok.Expiry.Unix() < time.Now().Unix() {
		return nil, token.ErrInvalidToken
	}

	return tok, err
}

// String returns basic
func (b *Basic) String() string {
	return "basic"
}
