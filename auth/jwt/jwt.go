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

// Generate a new auth ServiceAccount
func (s *svc) Generate(sa *auth.ServiceAccount) (*auth.ServiceAccount, error) {
	return nil, nil
}

// Revoke an authorization ServiceAccount
func (s *svc) Revoke(token string) error {
	return nil
}

// Validate a service account token
func (s *svc) Validate(token string) (*auth.ServiceAccount, error) {
	return nil, nil
}
