package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/auth"
	pb "github.com/micro/go-micro/v2/auth/service/proto"
	"github.com/micro/go-micro/v2/auth/token"
	"github.com/micro/go-micro/v2/auth/token/jwt"
	"github.com/micro/go-micro/v2/client"
	log "github.com/micro/go-micro/v2/logger"
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
	jwt     token.Provider
	rules   []*pb.Rule

	sync.Mutex
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

	// if we have a JWT public key passed as an option,
	// we can decode tokens with the type "JWT" locally
	// and not have to make an RPC call
	if key := s.options.PublicKey; len(key) > 0 {
		s.jwt = jwt.NewTokenProvider(token.WithPublicKey(key))
	}

	// load rules periodically from the auth service
	timer := time.NewTicker(time.Second * 30)
	go func() {
		for {
			s.loadRules()
			<-timer.C
		}
	}()
}

func (s *svc) Options() auth.Options {
	return s.options
}

// Generate a new account
func (s *svc) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	options := auth.NewGenerateOptions(opts...)

	rsp, err := s.auth.Generate(context.TODO(), &pb.GenerateRequest{
		Id:           id,
		Roles:        options.Roles,
		Metadata:     options.Metadata,
		SecretExpiry: int64(options.SecretExpiry.Nanoseconds()),
	})
	if err != nil {
		return nil, err
	}

	return serializeAccount(rsp.Account), nil
}

// Grant access to a resource
func (s *svc) Grant(role string, res *auth.Resource) error {
	_, err := s.auth.Grant(context.TODO(), &pb.GrantRequest{
		Role: role,
		Resource: &pb.Resource{
			Type:     res.Type,
			Name:     res.Name,
			Endpoint: res.Endpoint,
		},
	})
	return err
}

// Revoke access to a resource
func (s *svc) Revoke(role string, res *auth.Resource) error {
	_, err := s.auth.Revoke(context.TODO(), &pb.RevokeRequest{
		Role: role,
		Resource: &pb.Resource{
			Type:     res.Type,
			Name:     res.Name,
			Endpoint: res.Endpoint,
		},
	})
	return err
}

// Verify an account has access to a resource
func (s *svc) Verify(acc *auth.Account, res *auth.Resource) error {
	queries := [][]string{
		{res.Type, "*"},                         // check for wildcard resource type, e.g. service.*
		{res.Type, res.Name, "*"},               // check for wildcard name, e.g. service.foo*
		{res.Type, res.Name, res.Endpoint, "*"}, // check for wildcard endpoints, e.g. service.foo.ListFoo:*
		{res.Type, res.Name, res.Endpoint},      // check for specific role, e.g. service.foo.ListFoo:admin
	}

	// endpoint is a url which can have wildcard excludes, e.g.
	// "/foo/*" will allow "/foo/bar"
	if comps := strings.Split(res.Endpoint, "/"); len(comps) > 1 {
		for i := 1; i < len(comps); i++ {
			wildcard := fmt.Sprintf("%v/*", strings.Join(comps[0:i], "/"))
			queries = append(queries, []string{res.Type, res.Name, wildcard})
		}
	}

	for _, q := range queries {
		for _, rule := range s.listRules(q...) {
			if isValidRule(rule, acc, res) {
				return nil
			}
		}
	}

	return auth.ErrForbidden
}

// Inspect a token
func (s *svc) Inspect(token string) (*auth.Account, error) {
	// try to decode JWT locally and fall back to srv if an error
	// occurs, TODO: find a better way of determining if the token
	// is a JWT, possibly update the interface to take an auth.Token
	// and not just the string
	if len(strings.Split(token, ".")) == 3 && s.jwt != nil {
		if tok, err := s.jwt.Inspect(token); err == nil {
			return &auth.Account{
				ID:       tok.Subject,
				Roles:    tok.Roles,
				Metadata: tok.Metadata,
			}, nil
		}
	}

	rsp, err := s.auth.Inspect(context.TODO(), &pb.InspectRequest{
		Token: token,
	})
	if err != nil {
		return nil, err
	}

	return serializeAccount(rsp.Account), nil
}

// Refresh an account using a secret
func (s *svc) Refresh(secret string, opts ...auth.RefreshOption) (*auth.Token, error) {
	options := auth.NewRefreshOptions(opts...)

	rsp, err := s.auth.Refresh(context.Background(), &pb.RefreshRequest{
		Secret:      secret,
		TokenExpiry: int64(options.TokenExpiry.Seconds()),
	})
	if err != nil {
		return nil, err
	}

	return serializeToken(rsp.Token), nil
}

var ruleJoinKey = ":"

// isValidRule returns a bool, indicating if a rule permits access to a
// resource for a given account
func isValidRule(rule *pb.Rule, acc *auth.Account, res *auth.Resource) bool {
	if rule.Role == "*" {
		return true
	}

	for _, role := range acc.Roles {
		if rule.Role == role {
			return true
		}

		// allow user.anything if role is user.*
		if strings.HasSuffix(rule.Role, ".*") && strings.HasPrefix(rule.Role, role+".") {
			return true
		}
	}

	return false
}

// listRules gets all the rules from the store which have an id
// prefix matching the filters
func (s *svc) listRules(filters ...string) []*pb.Rule {
	s.Lock()
	defer s.Unlock()

	prefix := strings.Join(filters, ruleJoinKey)

	var rules []*pb.Rule
	for _, r := range s.rules {
		if strings.HasPrefix(r.Id, prefix) {
			rules = append(rules, r)
		}
	}

	return rules
}

// loadRules retrieves the rules from the auth service
func (s *svc) loadRules() {
	rsp, err := s.auth.ListRules(context.TODO(), &pb.ListRulesRequest{}, client.WithRetries(3))
	s.Lock()
	defer s.Unlock()

	if err != nil {
		log.Errorf("Error listing rules: %v", err)
		s.rules = []*pb.Rule{}
		return
	}

	s.rules = rsp.Rules
}

func serializeToken(t *pb.Token) *auth.Token {
	return &auth.Token{
		Token:    t.Token,
		Type:     t.Type,
		Created:  time.Unix(t.Created, 0),
		Expiry:   time.Unix(t.Expiry, 0),
		Subject:  t.Subject,
		Roles:    t.Roles,
		Metadata: t.Metadata,
	}
}

func serializeAccount(a *pb.Account) *auth.Account {
	var secret *auth.Token
	if a.Secret != nil {
		secret = serializeToken(a.Secret)
	}

	return &auth.Account{
		ID:       a.Id,
		Roles:    a.Roles,
		Metadata: a.Metadata,
		Secret:   secret,
	}
}
