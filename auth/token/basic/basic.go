package basic

import (
	"encoding/json"
	"fmt"
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

var (
	// StorePrefix to isolate tokens
	StorePrefix = "tokens/"
)

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
func (b *Basic) Generate(acc *auth.Account, opts ...token.GenerateOption) (*token.Token, error) {
	options := token.NewGenerateOptions(opts...)

	// marshal the account to bytes
	bytes, err := json.Marshal(acc)
	if err != nil {
		return nil, err
	}

	// write to the store
	key := uuid.New().String()
	err = b.store.Write(&store.Record{
		Key:    fmt.Sprintf("%v%v", StorePrefix, key),
		Value:  bytes,
		Expiry: options.Expiry,
	})
	if err != nil {
		return nil, err
	}

	// return the token
	return &token.Token{
		Token:   key,
		Created: time.Now(),
		Expiry:  time.Now().Add(options.Expiry),
	}, nil
}

// Inspect a token
func (b *Basic) Inspect(t string) (*auth.Account, error) {
	// lookup the token in the store
	recs, err := b.store.Read(StorePrefix + t)
	if err == store.ErrNotFound {
		return nil, token.ErrInvalidToken
	} else if err != nil {
		return nil, err
	}
	bytes := recs[0].Value

	// unmarshal the bytes
	var acc *auth.Account
	if err := json.Unmarshal(bytes, &acc); err != nil {
		return nil, err
	}

	return acc, nil
}

// String returns basic
func (b *Basic) String() string {
	return "basic"
}
