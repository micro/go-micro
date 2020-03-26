package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/auth"
	authPb "github.com/micro/go-micro/v2/auth/service/proto/auth"
	rulePb "github.com/micro/go-micro/v2/auth/service/proto/rules"
	"github.com/micro/go-micro/v2/auth/token"
	"github.com/micro/go-micro/v2/auth/token/jwt"
	"github.com/micro/go-micro/v2/client"
	log "github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/util/jitter"
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
	auth    authPb.AuthService
	rule    rulePb.RulesService
	jwt     token.Provider

	rules []*rulePb.Rule
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
	s.auth = authPb.NewAuthService("go.micro.auth", dc)
	s.rule = rulePb.NewRulesService("go.micro.auth", dc)

	// if we have a JWT public key passed as an option,
	// we can decode tokens with the type "JWT" locally
	// and not have to make an RPC call
	if key := s.options.PublicKey; len(key) > 0 {
		s.jwt = jwt.NewTokenProvider(token.WithPublicKey(key))
	}

	// load rules periodically from the auth service
	timer := time.NewTicker(time.Second * 30)
	go func() {
		// load rules immediately on startup
		s.loadRules()

		for {
			<-timer.C

			// jitter for up to 5 seconds, this stops
			// all the services calling the auth service
			// at the exact same time
			time.Sleep(jitter.Do(time.Second * 5))
			s.loadRules()
		}
	}()
}

func (s *svc) Options() auth.Options {
	return s.options
}

// Generate a new account
func (s *svc) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	options := auth.NewGenerateOptions(opts...)

	rsp, err := s.auth.Generate(context.TODO(), &authPb.GenerateRequest{
		Id:           id,
		Roles:        options.Roles,
		Metadata:     options.Metadata,
		SecretExpiry: int64(options.SecretExpiry.Seconds()),
	})
	if err != nil {
		return nil, err
	}

	return serializeAccount(rsp.Account), nil
}

// Grant access to a resource
func (s *svc) Grant(role string, res *auth.Resource) error {
	_, err := s.rule.Create(context.TODO(), &rulePb.CreateRequest{
		Role:   role,
		Access: rulePb.Access_GRANTED,
		Resource: &authPb.Resource{
			Type:     res.Type,
			Name:     res.Name,
			Endpoint: res.Endpoint,
		},
	})
	return err
}

// Revoke access to a resource
func (s *svc) Revoke(role string, res *auth.Resource) error {
	_, err := s.rule.Delete(context.TODO(), &rulePb.DeleteRequest{
		Role:   role,
		Access: rulePb.Access_GRANTED,
		Resource: &authPb.Resource{
			Type:     res.Type,
			Name:     res.Name,
			Endpoint: res.Endpoint,
		},
	})
	return err
}

// Verify an account has access to a resource
func (s *svc) Verify(acc *auth.Account, res *auth.Resource) error {
	log.Infof("%v requesting access to %v:%v:%v", acc.ID, res.Type, res.Name, res.Endpoint)

	queries := [][]string{
		{res.Type, res.Name, res.Endpoint}, // check for specific role, e.g. service.foo.ListFoo:admin (role is checked in accessForRule)
		{res.Type, res.Name, "*"},          // check for wildcard endpoint, e.g. service.foo*
		{res.Type, "*"},                    // check for wildcard name, e.g. service.*
		{"*"},                              // check for wildcard type, e.g. *
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
			switch accessForRule(rule, acc, res) {
			case rulePb.Access_UNKNOWN:
				continue // rule did not specify access, check the next rule
			case rulePb.Access_GRANTED:
				log.Infof("%v granted access to %v:%v:%v by rule %v", acc.ID, res.Type, res.Name, res.Endpoint, rule.Id)
				return nil // rule grants the account access to the resource
			case rulePb.Access_DENIED:
				log.Infof("%v denied access to %v:%v:%v by rule %v", acc.ID, res.Type, res.Name, res.Endpoint, rule.Id)
				return auth.ErrForbidden // rule denies access to the resource
			}
		}
	}

	// no rules were found for the resource, default to denying access
	log.Infof("%v denied access to %v:%v:%v by lack of rule", acc.ID, res.Type, res.Name, res.Endpoint)
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

	rsp, err := s.auth.Inspect(context.TODO(), &authPb.InspectRequest{
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

	rsp, err := s.auth.Refresh(context.Background(), &authPb.RefreshRequest{
		Secret:      secret,
		TokenExpiry: int64(options.TokenExpiry.Seconds()),
	})
	if err != nil {
		return nil, err
	}

	return serializeToken(rsp.Token), nil
}

var ruleJoinKey = ":"

// accessForRule returns a rule status, indicating if a rule permits access to a
// resource for a given account
func accessForRule(rule *rulePb.Rule, acc *auth.Account, res *auth.Resource) rulePb.Access {
	if rule.Role == "*" {
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

	return rulePb.Access_UNKNOWN
}

// listRules gets all the rules from the store which have an id
// prefix matching the filters
func (s *svc) listRules(filters ...string) []*rulePb.Rule {
	s.Lock()
	defer s.Unlock()

	prefix := strings.Join(filters, ruleJoinKey)

	var rules []*rulePb.Rule
	for _, r := range s.rules {
		if strings.HasPrefix(r.Id, prefix) {
			rules = append(rules, r)
		}
	}

	return rules
}

// loadRules retrieves the rules from the auth service
func (s *svc) loadRules() {
	log.Infof("Loading rules from auth service")
	rsp, err := s.rule.List(context.TODO(), &rulePb.ListRequest{})
	s.Lock()
	defer s.Unlock()

	if err != nil {
		log.Errorf("Error listing rules: %v", err)
		return
	}

	log.Infof("Loaded %v rules from the auth service", len(rsp.Rules))
	s.rules = rsp.Rules
}

func serializeToken(t *authPb.Token) *auth.Token {
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

func serializeAccount(a *authPb.Account) *auth.Account {
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
