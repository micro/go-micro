package service

import (
	"context"
	"time"

	"github.com/micro/go-micro/auth"
	pb "github.com/micro/go-micro/auth/service/proto"
	"github.com/micro/go-micro/client"
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

func (s *svc) Init(opts ...auth.Option) error {
	for _, o := range opts {
		o(&s.options)
	}

	dc := client.DefaultClient
	s.auth = pb.NewAuthService("go.micro.auth", dc)

	return nil
}

// Generate a new auth account
func (s *svc) Generate(sa *auth.Account) (*auth.Account, error) {
	// construct the request
	req := &pb.GenerateRequest{Account: serializeAccount(sa)}

	// execute the request
	resp, err := s.auth.Generate(context.Background(), req)
	if err != nil {
		return nil, err
	}

	// format the response
	return deserializeAccount(resp.Account), nil
}

// Revoke an authorization account
func (s *svc) Revoke(token string) error {
	// contruct the request
	req := &pb.RevokeRequest{Token: token}

	// execute the request
	_, err := s.auth.Revoke(context.Background(), req)
	return err
}

// Validate an account token
func (s *svc) Validate(token string) (*auth.Account, error) {
	resp, err := s.auth.Validate(context.Background(), &pb.ValidateRequest{Token: token})
	if err != nil {
		return nil, err
	}

	return deserializeAccount(resp.Account), nil
}

func serializeAccount(sa *auth.Account) *pb.Account {
	roles := make([]*pb.Role, len(sa.Roles))
	for i, r := range sa.Roles {
		roles[i] = &pb.Role{
			Name: r.Name,
		}

		if r.Resource != nil {
			roles[i].Resource = &pb.Resource{
				Name: r.Resource.Name,
				Type: r.Resource.Type,
			}
		}
	}

	return &pb.Account{
		Roles:    roles,
		Metadata: sa.Metadata,
		Parent: &pb.Resource{
			Name: sa.Parent.Name,
			Type: sa.Parent.Type,
		},
	}
}

func deserializeAccount(a *pb.Account) *auth.Account {
	// format the response
	sa := &auth.Account{
		Token:    a.Token,
		Created:  time.Unix(a.Created, 0),
		Expiry:   time.Unix(a.Expiry, 0),
		Metadata: a.Metadata,
	}
	if a.Parent != nil {
		sa.Parent = &auth.Resource{
			Name: a.Parent.Name,
			Type: a.Parent.Type,
		}
	}

	sa.Roles = make([]*auth.Role, len(a.Roles))
	for i, r := range a.Roles {
		sa.Roles[i] = &auth.Role{
			Name: r.Name,
		}

		if r.Resource != nil {
			sa.Roles[i].Resource = &auth.Resource{
				Name: r.Resource.Name,
				Type: r.Resource.Type,
			}
		}
	}

	return sa
}
