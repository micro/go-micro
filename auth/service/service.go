package service

import (
	"github.com/micro/go-micro/v2/auth"
	pb "github.com/micro/go-micro/v2/auth/service/proto"
	"github.com/micro/go-micro/v2/auth/token"
	"github.com/micro/go-micro/v2/client"
)

// NewAuth returns a new instance of the Auth service
func NewAuth(opts ...auth.Option) auth.Auth {
	svc := new(svc)
	svc.Init(opts...)
	return svc
}

// svc is the service implementation of the Auth interface
type svc struct {
	options auth.Options
	auth    pb.AuthService
}

func (s *svc) String() string {
	return "service"
}

func (s *svc) Init(opts ...auth.Option) {
	for _, o := range opts {
		o(&s.options)
	}

	dc := client.DefaultClient
	s.auth = pb.NewAuthService("go.micro.auth", dc)
}

func (s *svc) Options() auth.Options {
	return s.options
}

// Generate a new account
func (s *svc) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	return nil, nil
}

// Grant access to a resource
func (s *svc) Grant(role string, res *auth.Resource) error {
	return nil
}

// Revoke access to a resource
func (s *svc) Revoke(role string, res *auth.Resource) error {
	return nil
}

// Verify an account has access to a resource
func (s *svc) Verify(acc *auth.Account, res *auth.Resource) error {
	return nil
}

// Inspect a token
func (s *svc) Inspect(token string) (*auth.Account, error) {
	return nil, nil
}

// Refresh an account using a secret
func (s *svc) Refresh(secret string) (*token.Token, error) {
	return nil, nil
}
