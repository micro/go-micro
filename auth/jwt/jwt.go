package jwt

import (
	"log"

	"github.com/micro/go-micro/auth"
)

// NewAuth returns a new instance of the Auth service
func NewAuth(opts ...auth.Option) auth.Auth {
	svc := new(svc)
	svc.Init(opts...)
	return svc
}

// svc is the JWT implementation of the Auth interface
type svc struct {
	options auth.Options
}

func (s *svc) Init(opts ...auth.Option) error {
	for _, o := range opts {
		o(&s.options)
	}

	if s.options.PublicKey == "" {
		log.Fatal("Auth: MICRO_AUTH_PUBLIC_KEY is blank")
	}

	return nil
}

// Generate a new auth account
func (s *svc) Generate(sa *auth.Account) (*auth.Account, error) {
	return nil, nil
}

// Revoke an authorization account
func (s *svc) Revoke(token string) error {
	return nil
}

// Validate an account token
func (s *svc) Validate(token string) (*auth.Account, error) {
	return nil, nil
}
