package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/metadata"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/auth/rules"
	pb "github.com/micro/go-micro/v2/auth/service/proto"
	"github.com/micro/go-micro/v2/auth/token"
	"github.com/micro/go-micro/v2/auth/token/jwt"
	"github.com/micro/go-micro/v2/client"
	log "github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/util/jitter"
)

// svc is the service implementation of the Auth interface
type svc struct {
	options auth.Options
	auth    pb.AuthService
	rule    pb.RulesService
	jwt     token.Provider
	rules   map[string][]*auth.Rule
	sync.Mutex
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
	s.rule = pb.NewRulesService("go.micro.auth", s.options.Client)

	// if we have a JWT public key passed as an option,
	// we can decode tokens with the type "JWT" locally
	// and not have to make an RPC call
	if key := s.options.PublicKey; len(key) > 0 {
		s.jwt = jwt.NewTokenProvider(token.WithPublicKey(key))
	}
}

func (s *svc) Options() auth.Options {
	s.Lock()
	defer s.Unlock()
	return s.options
}

// Generate a new account
func (s *svc) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	options := auth.NewGenerateOptions(opts...)

	rsp, err := s.auth.Generate(context.TODO(), &pb.GenerateRequest{
		Id:       id,
		Type:     options.Type,
		Secret:   options.Secret,
		Roles:    options.Roles,
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
	_, err := s.rule.Create(context.TODO(), &pb.CreateRequest{
		Rule: &pb.Rule{
			Id:       rule.ID,
			Role:     rule.Role,
			Priority: rule.Priority,
			Access:   pb.Access_GRANTED,
			Resource: &pb.Resource{
				Type:     rule.Resource.Type,
				Name:     rule.Resource.Name,
				Endpoint: rule.Resource.Endpoint,
			},
		},
	})
	go s.loadRules(s.options.Namespace)
	return err
}

// Revoke access to a resource
func (s *svc) Revoke(rule *auth.Rule) error {
	_, err := s.rule.Delete(context.TODO(), &pb.DeleteRequest{
		Id: rule.ID,
	})
	go s.loadRules(s.options.Namespace)
	return err
}

func (s *svc) Rules() ([]*auth.Rule, error) {
	return s.rules[s.options.Namespace], nil
}

// Verify an account has access to a resource
func (s *svc) Verify(acc *auth.Account, res *auth.Resource, opts ...auth.VerifyOption) error {
	options := auth.VerifyOptions{Scope: s.options.Namespace}
	for _, o := range opts {
		o(&options)
	}

	// load the rules if none are loaded
	s.loadRulesIfEmpty(options.Scope)

	// verify the request using the rules
	return rules.Verify(options.Scope, s.rules[options.Scope], acc, res)
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

var ruleJoinKey = ":"

// accessForRule returns a rule status, indicating if a rule permits access to a
// resource for a given account
func accessForRule(rule *pb.Rule, acc *auth.Account, res *auth.Resource) pb.Access {
	// a blank role permits access to the public
	if rule.Role == "" {
		return rule.Access
	}

	// a * role permits access to any user
	if rule.Role == "*" && acc != nil {
		return rule.Access
	}

	for _, role := range acc.Roles {
		if rule.Role == role {
			return rule.Access
		}

		// allow user.anything if role is user.*
		if strings.HasSuffix(rule.Role, ".*") && strings.HasPrefix(rule.Role, role+".") {
			return rule.Access
		}
	}

	return pb.Access_UNKNOWN
}

// loadRules retrieves the rules from the auth service. Since this implementation is used by micro
// clients, which support muti-tenancy we may have to persist rules in multiple namespaces.
func (s *svc) loadRules(namespace string) {
	ctx := metadata.Set(context.TODO(), "Micro-Namespace", namespace)
	rsp, err := s.rule.List(ctx, &pb.ListRequest{})
	if err != nil {
		log.Errorf("Error listing rules: %v", err)
		return
	}

	rules := make([]*auth.Rule, 0, len(rsp.Rules))
	for _, r := range rsp.Rules {
		var access auth.Access
		if r.Access == pb.Access_GRANTED {
			access = auth.AccessGranted
		} else {
			access = auth.AccessDenied
		}

		rules = append(rules, &auth.Rule{
			ID:       r.Id,
			Role:     r.Role,
			Access:   access,
			Priority: r.Priority,
			Resource: &auth.Resource{
				Type:     r.Resource.Type,
				Name:     r.Resource.Name,
				Endpoint: r.Resource.Endpoint,
			},
		})
	}

	s.Lock()
	s.rules[namespace] = rules
	s.Unlock()
}

func (s *svc) loadRulesIfEmpty(namespace string) {
	s.Lock()
	rules := s.rules
	s.Unlock()

	if _, ok := rules[namespace]; !ok {
		s.loadRules(namespace)
	}
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
		Roles:    a.Roles,
		Secret:   a.Secret,
		Metadata: a.Metadata,
		Provider: a.Provider,
		Scopes:   a.Scopes,
	}
}

// NewAuth returns a new instance of the Auth service
func NewAuth(opts ...auth.Option) auth.Auth {
	options := auth.NewOptions(opts...)
	if options.Client == nil {
		options.Client = client.DefaultClient
	}

	service := &svc{
		auth:    pb.NewAuthService("go.micro.auth", options.Client),
		rule:    pb.NewRulesService("go.micro.auth", options.Client),
		rules:   make(map[string][]*auth.Rule),
		options: options,
	}

	// load rules periodically from the auth service
	go func() {
		ruleTimer := time.NewTicker(time.Second * 30)

		for {
			<-ruleTimer.C
			time.Sleep(jitter.Do(time.Second * 5))

			for ns := range service.rules {
				service.loadRules(ns)
			}
		}
	}()

	return service
}
