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
	auth    pb.AuthService
	rule    pb.RulesService
	jwt     token.Provider

	rules []*pb.Rule
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
	s.rule = pb.NewRulesService("go.micro.auth", dc)

	// if we have a JWT public key passed as an option,
	// we can decode tokens with the type "JWT" locally
	// and not have to make an RPC call
	if key := s.options.PublicKey; len(key) > 0 {
		s.jwt = jwt.NewTokenProvider(token.WithPublicKey(key))
	}

	// load rules periodically from the auth service
	ruleTimer := time.NewTicker(time.Second * 30)
	go func() {
		// load rules immediately on startup
		s.loadRules()

		for {
			<-ruleTimer.C

			// jitter for up to 5 seconds, this stops
			// all the services calling the auth service
			// at the exact same time
			time.Sleep(jitter.Do(time.Second * 5))
			s.loadRules()
		}
	}()

	// we have client credentials and must load a new token
	// periodically
	if len(s.options.ID) > 0 || len(s.options.RefreshToken) > 0 {
		tokenTimer := time.NewTicker(time.Minute)

		go func() {
			s.loadToken()

			for {
				<-tokenTimer.C

				// Do not get a new token if the current one has more than three
				// minutes remaining. We do 3 minutes to allow multiple retires in
				// the case one request fails
				t := s.Options().Token
				if t != nil && t.Expiry.Unix() > time.Now().Add(time.Minute*3).Unix() {
					continue
				}

				// jitter for up to 5 seconds, this stops
				// all the services calling the auth service
				// at the exact same time
				time.Sleep(jitter.Do(time.Second * 5))
				s.loadToken()
			}
		}()
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
		Id:        id,
		Type:      options.Type,
		Roles:     options.Roles,
		Secret:    options.Secret,
		Metadata:  options.Metadata,
		Provider:  options.Provider,
		Namespace: options.Namespace,
	})
	if err != nil {
		return nil, err
	}

	return serializeAccount(rsp.Account), nil
}

// Login to an account
func (s *svc) Login(id string, opts ...auth.LoginOption) (*auth.Account, error) {
	options := auth.NewLoginOptions(opts...)
	rsp, err := s.auth.Login(context.TODO(), &pb.LoginRequest{Id: id, Secret: options.Secret})
	if err != nil {
		return nil, err
	}
	return serializeAccount(rsp.Account), nil
}

// Grant access to a resource
func (s *svc) Grant(role string, res *auth.Resource) error {
	_, err := s.rule.Create(context.TODO(), &pb.CreateRequest{
		Role:   role,
		Access: pb.Access_GRANTED,
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
	_, err := s.rule.Delete(context.TODO(), &pb.DeleteRequest{
		Role:   role,
		Access: pb.Access_GRANTED,
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
			case pb.Access_UNKNOWN:
				continue // rule did not specify access, check the next rule
			case pb.Access_GRANTED:
				log.Infof("%v granted access to %v:%v:%v by rule %v", acc.ID, res.Type, res.Name, res.Endpoint, rule.Id)
				return nil // rule grants the account access to the resource
			case pb.Access_DENIED:
				log.Infof("%v denied access to %v:%v:%v by rule %v", acc.ID, res.Type, res.Name, res.Endpoint, rule.Id)
				return auth.ErrForbidden // rule denies access to the resource
			}
		}
	}

	// no rules were found for the resource, default to denying access
	log.Infof("%v denied access to %v:%v:%v by lack of rule (%v rules found)", acc.ID, res.Type, res.Name, res.Endpoint, len(s.rules))
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

// Token generation using an account ID and secret
func (s *svc) Token(id, refresh string, opts ...auth.TokenOption) (*auth.Token, error) {
	options := auth.NewTokenOptions(opts...)

	rsp, err := s.auth.Token(context.Background(), &pb.TokenRequest{
		Id:           id,
		RefreshToken: refresh,
		TokenExpiry:  int64(options.TokenExpiry.Seconds()),
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

	return pb.Access_UNKNOWN
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
	rsp, err := s.rule.List(context.TODO(), &pb.ListRequest{})
	s.Lock()
	defer s.Unlock()

	if err != nil {
		log.Errorf("Error listing rules: %v", err)
		return
	}

	s.rules = rsp.Rules
}

// loadToken generates a new token for the service to use when making calls
func (s *svc) loadToken() {
	rsp, err := s.auth.Token(context.TODO(), &pb.TokenRequest{
		Id:           s.Options().ID,
		RefreshToken: s.Options().RefreshToken,
		TokenExpiry:  int64((time.Minute * 15).Seconds()),
	})
	s.Lock()
	defer s.Unlock()

	if err != nil {
		log.Errorf("Error generating token: %v", err)
		return
	}

	s.options.Token = serializeToken(rsp.Token)
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
	return &auth.Account{
		ID:           a.Id,
		Roles:        a.Roles,
		Metadata:     a.Metadata,
		Provider:     a.Provider,
		Namespace:    a.Namespace,
		RefreshToken: a.RefreshToken,
	}
}
