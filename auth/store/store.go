package store

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/store"
)

type Auth struct {
	store store.Store
	opts  auth.Options
}

// NewAuth returns an instance of store auth
func NewAuth(opts ...auth.Option) auth.Auth {
	var options auth.Options

	for _, o := range opts {
		o(&options)
	}

	return &Auth{
		store: store.DefaultStore,
		opts:  options,
	}
}

// Init the auth package
func (a *Auth) Init(opts ...auth.Option) error {
	for _, o := range opts {
		o(&a.opts)
	}
	return nil
}

// Options returns the options set
func (a *Auth) Options() auth.Options {
	return a.opts
}

// Generate a new auth Account
func (a *Auth) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	// generate the token
	token, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	// parse the options
	options := auth.NewGenerateOptions(opts...)

	// construct the account
	sa := auth.Account{
		Id:       id,
		Token:    token.String(),
		Created:  time.Now(),
		Metadata: options.Metadata,
		Roles:    options.Roles,
	}

	// encode the data to bytes
	// TODO: replace with json
	buf := &bytes.Buffer{}
	e := gob.NewEncoder(buf)
	if err := e.Encode(sa); err != nil {
		return nil, err
	}

	// write to the store
	err = a.store.Write(&store.Record{
		Key:   token.String(),
		Value: buf.Bytes(),
	})
	if err != nil {
		return nil, err
	}

	// return the result
	return &sa, nil
}

// Revoke an authorization Account
func (a *Auth) Revoke(token string) error {
	records, err := a.store.Read(token, store.ReadSuffix())
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return errors.BadRequest("go.micro.auth", "token not found")
	}

	for _, r := range records {
		if err := a.store.Delete(r.Key); err != nil {
			return errors.InternalServerError("go.micro.auth", "error deleting from store")
		}
	}

	return nil
}

// Verify an account token
func (a *Auth) Verify(token string) (*auth.Account, error) {
	// lookup the record by token
	records, err := a.store.Read(token, store.ReadSuffix())
	if err == store.ErrNotFound || len(records) == 0 {
		return nil, errors.Unauthorized("go.micro.auth", "invalid token")
	} else if err != nil {
		return nil, errors.InternalServerError("go.micro.auth", "error reading store")
	}

	// decode the result
	// TODO: replace with json
	b := bytes.NewBuffer(records[0].Value)
	decoder := gob.NewDecoder(b)
	var sa auth.Account
	err = decoder.Decode(&sa)

	// return the result
	return &sa, err
}

// String returns the implementation
func (a *Auth) String() string {
	return "store"
}
