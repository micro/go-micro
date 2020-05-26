package service

import (
	"context"
	"strings"
	"time"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/auth/rules"
	pb "github.com/micro/go-micro/v2/auth/service/proto"
	"github.com/micro/go-micro/v2/auth/token"
	"github.com/micro/go-micro/v2/auth/token/jwt"
	"github.com/micro/go-micro/v2/client"
)

// svc is the service implementation of the Auth interface
type svc struct {
	options auth.Options
	auth    pb.AuthService
	rules   pb.RulesService
	jwt     token.Provider
}

func (s *svc) String() string {
	return "service"
}

func (s *svc) Init(opts ...auth.Option) {
	for _, o := range opts {
		o(&s.options)
	}

	if s.options.Client == nil {
		s.options.Client = client.DefaultClient
	}

	s.auth = pb.NewAuthService("go.micro.auth", s.options.Client)
	s.rules = pb.NewRulesService("go.micro.auth", s.options.Client)

	// if we have a JWT public key passed as an option,
	// we can decode tokens with the type "JWT" locally
	// and not have to make an RPC call
	if key := s.options.PublicKey; len(key) > 0 {
		s.jwt = jwt.NewTokenProvider(token.WithPublicKey(key))
	}
}

func (s *svc) Options() auth.Options {
	return s.options
}

// Generate a new account
func (s *svc) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	options := auth.NewGenerateOptions(opts...)

	rsp, err := s.auth.Generate(context.TODO(), &pb.GenerateRequest{
		Id:       id,
		Type:     options.Type,
		Secret:   options.Secret,
		Scopes:   options.Scopes,
		Metadata: options.Metadata,
		Provider: options.Provider,
	})
	if err != nil {
		return nil, err
	}

	return serializeAccount(rsp.Account), nil
}

// Grant access to a resource
func (s *svc) Grant(rule *auth.Rule) error {
	access := pb.Access_UNKNOWN
	if rule.Access == auth.AccessGranted {
		access = pb.Access_GRANTED
	} else if rule.Access == auth.AccessDenied {
		access = pb.Access_DENIED
	}

	_, err := s.rules.Create(context.TODO(), &pb.CreateRequest{
		Rule: &pb.Rule{
			Id:       rule.ID,
			Scope:    rule.Scope,
			Priority: rule.Priority,
			Access:   access,
			Resource: &pb.Resource{
				Type:     rule.Resource.Type,
				Name:     rule.Resource.Name,
				Endpoint: rule.Resource.Endpoint,
			},
		},
	})

	return err
}

// Revoke access to a resource
func (s *svc) Revoke(rule *auth.Rule) error {
	_, err := s.rules.Delete(context.TODO(), &pb.DeleteRequest{
		Id: rule.ID,
	})

	return err
}

func (s *svc) Rules(opts ...auth.RulesOption) ([]*auth.Rule, error) {
	var options auth.RulesOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Context == nil {
		options.Context = context.TODO()
	}

	rsp, err := s.rules.List(options.Context, &pb.ListRequest{}, client.WithCache(time.Second*30))
	if err != nil {
		return nil, err
	}

	rules := make([]*auth.Rule, len(rsp.Rules))
	for i, r := range rsp.Rules {
		rules[i] = serializeRule(r)
	}

	return rules, nil
}

// Verify an account has access to a resource
func (s *svc) Verify(acc *auth.Account, res *auth.Resource, opts ...auth.VerifyOption) error {
	var options auth.VerifyOptions
	for _, o := range opts {
		o(&options)
	}

	rs, err := s.Rules(auth.RulesContext(options.Context))
	if err != nil {
		return err
	}

	return rules.Verify(rs, acc, res)
}

// Inspect a token
func (s *svc) Inspect(token string) (*auth.Account, error) {
	// try to decode JWT locally and fall back to srv if an error occurs
	if len(strings.Split(token, ".")) == 3 && s.jwt != nil {
		return s.jwt.Inspect(token)
	}

	// the token is not a JWT or we do not have the keys to decode it,
	// fall back to the auth service
	rsp, err := s.auth.Inspect(context.TODO(), &pb.InspectRequest{Token: token})
	if err != nil {
		return nil, err
	}
	return serializeAccount(rsp.Account), nil
}

// Token generation using an account ID and secret
func (s *svc) Token(opts ...auth.TokenOption) (*auth.Token, error) {
	options := auth.NewTokenOptions(opts...)

	rsp, err := s.auth.Token(context.Background(), &pb.TokenRequest{
		Id:           options.ID,
		Secret:       options.Secret,
		RefreshToken: options.RefreshToken,
		TokenExpiry:  int64(options.Expiry.Seconds()),
	})
	if err != nil {
		return nil, err
	}

	return serializeToken(rsp.Token), nil
}

func serializeToken(t *pb.Token) *auth.Token {
	return &auth.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		Created:      time.Unix(t.Created, 0),
		Expiry:       time.Unix(t.Expiry, 0),
	}
}

func serializeAccount(a *pb.Account) *auth.Account {
	return &auth.Account{
		ID:       a.Id,
		Secret:   a.Secret,
		Issuer:   a.Issuer,
		Metadata: a.Metadata,
		Scopes:   a.Scopes,
	}
}

func serializeRule(r *pb.Rule) *auth.Rule {
	var access auth.Access
	if r.Access == pb.Access_GRANTED {
		access = auth.AccessGranted
	} else {
		access = auth.AccessDenied
	}

	return &auth.Rule{
		ID:       r.Id,
		Scope:    r.Scope,
		Access:   access,
		Priority: r.Priority,
		Resource: &auth.Resource{
			Type:     r.Resource.Type,
			Name:     r.Resource.Name,
			Endpoint: r.Resource.Endpoint,
		},
	}
}

// NewAuth returns a new instance of the Auth service
func NewAuth(opts ...auth.Option) auth.Auth {
	options := auth.NewOptions(opts...)
	if options.Client == nil {
		options.Client = client.DefaultClient
	}

	return &svc{
		auth:    pb.NewAuthService("go.micro.auth", options.Client),
		rules:   pb.NewRulesService("go.micro.auth", options.Client),
		options: options,
	}
}
