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

	s.auth = pb.NewAuthService("go.micro.auth", s.options.Client)
	s.rules = pb.NewRulesService("go.micro.auth", s.options.Client)

	s.setupJWT()
}

func (s *svc) Options() auth.Options {
	return s.options
}

// Generate a new account
func (s *svc) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	options := auth.NewGenerateOptions(opts...)

	// we have the JWT private key and generate ourselves an account
	if len(s.options.PrivateKey) > 0 {
		acc := &auth.Account{
			ID:       id,
			Type:     options.Type,
			Scopes:   options.Scopes,
			Metadata: options.Metadata,
			Issuer:   s.Options().Issuer,
		}

		tok, err := s.jwt.Generate(acc, token.WithExpiry(time.Hour*24*365))
		if err != nil {
			return nil, err
		}

		// when using JWTs, the account secret is the JWT's token. This
		// can be used as an argument in the Token method.
		acc.Secret = tok.Token
		return acc, nil
	}

	rsp, err := s.auth.Generate(context.TODO(), &pb.GenerateRequest{
		Id:       id,
		Type:     options.Type,
		Secret:   options.Secret,
		Scopes:   options.Scopes,
		Metadata: options.Metadata,
		Provider: options.Provider,
		Options: &pb.Options{
			Namespace: s.Options().Issuer,
		},
	}, s.callOpts()...)
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
		Options: &pb.Options{
			Namespace: s.Options().Issuer,
		},
	}, s.callOpts()...)

	return err
}

// Revoke access to a resource
func (s *svc) Revoke(rule *auth.Rule) error {
	_, err := s.rules.Delete(context.TODO(), &pb.DeleteRequest{
		Id: rule.ID, Options: &pb.Options{
			Namespace: s.Options().Issuer,
		},
	}, s.callOpts()...)

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
	if len(options.Namespace) == 0 {
		options.Namespace = s.options.Issuer
	}

	callOpts := append(s.callOpts(), client.WithCache(time.Second*30))
	rsp, err := s.rules.List(options.Context, &pb.ListRequest{
		Options: &pb.Options{Namespace: options.Namespace},
	}, callOpts...)
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

	rs, err := s.Rules(
		auth.RulesContext(options.Context),
		auth.RulesNamespace(options.Namespace),
	)
	if err != nil {
		return err
	}

	return rules.Verify(rs, acc, res)
}

// Inspect a token
func (s *svc) Inspect(token string) (*auth.Account, error) {
	// try to decode JWT locally and fall back to srv if an error occurs
	if len(strings.Split(token, ".")) == 3 && len(s.options.PublicKey) > 0 {
		return s.jwt.Inspect(token)
	}

	// the token is not a JWT or we do not have the keys to decode it,
	// fall back to the auth service
	rsp, err := s.auth.Inspect(context.TODO(), &pb.InspectRequest{
		Token: token, Options: &pb.Options{Namespace: s.Options().Issuer},
	}, s.callOpts()...)
	if err != nil {
		return nil, err
	}
	return serializeAccount(rsp.Account), nil
}

// Token generation using an account ID and secret
func (s *svc) Token(opts ...auth.TokenOption) (*auth.Token, error) {
	options := auth.NewTokenOptions(opts...)

	// we have the JWT private key and refresh accounts locally
	if len(s.options.PrivateKey) > 0 {
		tok := options.RefreshToken
		if len(options.Secret) > 0 {
			tok = options.Secret
		}

		acc, err := s.jwt.Inspect(tok)
		if err != nil {
			return nil, err
		}

		token, err := s.jwt.Generate(acc, token.WithExpiry(options.Expiry))
		if err != nil {
			return nil, err
		}

		return &auth.Token{
			Expiry:       token.Expiry,
			AccessToken:  token.Token,
			RefreshToken: tok,
		}, nil
	}

	rsp, err := s.auth.Token(context.Background(), &pb.TokenRequest{
		Id:           options.ID,
		Secret:       options.Secret,
		RefreshToken: options.RefreshToken,
		TokenExpiry:  int64(options.Expiry.Seconds()),
		Options: &pb.Options{
			Namespace: s.Options().Issuer,
		},
	}, s.callOpts()...)
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

func (s *svc) callOpts() []client.CallOption {
	return []client.CallOption{
		client.WithAddress(s.options.Addrs...),
	}
}

// NewAuth returns a new instance of the Auth service
func NewAuth(opts ...auth.Option) auth.Auth {
	options := auth.NewOptions(opts...)
	if options.Client == nil {
		options.Client = client.DefaultClient
	}
	if len(options.Addrs) == 0 {
		options.Addrs = []string{"127.0.0.1:8010"}
	}

	service := &svc{
		auth:    pb.NewAuthService("go.micro.auth", options.Client),
		rules:   pb.NewRulesService("go.micro.auth", options.Client),
		options: options,
	}
	service.setupJWT()

	return service
}

func (s *svc) setupJWT() {
	tokenOpts := []token.Option{}

	// if we have a JWT public key passed as an option,
	// we can decode tokens with the type "JWT" locally
	// and not have to make an RPC call
	if key := s.options.PublicKey; len(key) > 0 {
		tokenOpts = append(tokenOpts, token.WithPublicKey(key))
	}

	// if we have a JWT private key passed as an option,
	// we can generate accounts locally and not have to make
	// an RPC call, this is used for micro clients such as
	// api, web, proxy.
	if key := s.options.PrivateKey; len(key) > 0 {
		tokenOpts = append(tokenOpts, token.WithPrivateKey(key))
	}

	s.jwt = jwt.NewTokenProvider(tokenOpts...)
}
