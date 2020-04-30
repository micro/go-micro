package service

import (
	"context"
	"fmt"
	"sort"
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
	return &svc{options: auth.NewOptions(opts...)}
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
	go func() {
		ruleTimer := time.NewTicker(time.Second * 30)

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
		Secret:    options.Secret,
		Roles:     options.Roles,
		Metadata:  options.Metadata,
		Provider:  options.Provider,
		Namespace: options.Namespace,
	})
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
			Namespace: res.Namespace,
			Type:      res.Type,
			Name:      res.Name,
			Endpoint:  res.Endpoint,
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
			Namespace: res.Namespace,
			Type:      res.Type,
			Name:      res.Name,
			Endpoint:  res.Endpoint,
		},
	})
	return err
}

// Verify an account has access to a resource
func (s *svc) Verify(acc *auth.Account, res *auth.Resource) error {
	// set the namespace on the resource
	if len(res.Namespace) == 0 {
		res.Namespace = s.Options().Namespace
	}

	queries := [][]string{
		{res.Namespace, res.Type, res.Name, res.Endpoint}, // check for specific role, e.g. service.foo.ListFoo:admin (role is checked in accessForRule)
		{res.Namespace, res.Type, res.Name, "*"},          // check for wildcard endpoint, e.g. service.foo*
		{res.Namespace, res.Type, "*"},                    // check for wildcard name, e.g. service.*
		{res.Namespace, "*"},                              // check for wildcard type, e.g. *
		{"*"},                                             // check for wildcard namespace
	}

	// endpoint is a url which can have wildcard excludes, e.g.
	// "/foo/*" will allow "/foo/bar"
	if comps := strings.Split(res.Endpoint, "/"); len(comps) > 1 {
		for i := 1; i < len(comps); i++ {
			wildcard := fmt.Sprintf("%v/*", strings.Join(comps[0:i], "/"))
			queries = append(queries, []string{res.Type, res.Name, wildcard})
		}
	}

	// set a default account id / namespace to log
	logID := acc.ID
	if len(logID) == 0 {
		logID = "[no account]"
	}
	logNamespace := acc.Namespace
	if len(logNamespace) == 0 {
		logNamespace = "[no namespace]"
	}

	for _, q := range queries {
		for _, rule := range s.listRules(q...) {
			switch accessForRule(rule, acc, res) {
			case pb.Access_UNKNOWN:
				continue // rule did not specify access, check the next rule
			case pb.Access_GRANTED:
				log.Tracef("%v:%v granted access to %v:%v:%v:%v by rule %v", logNamespace, logID, res.Namespace, res.Type, res.Name, res.Endpoint, rule.Id)
				return nil // rule grants the account access to the resource
			case pb.Access_DENIED:
				log.Tracef("%v:%v denied access to %v:%v:%v:%v by rule %v", logNamespace, logID, res.Namespace, res.Type, res.Name, res.Endpoint, rule.Id)
				return auth.ErrForbidden // rule denies access to the resource
			}
		}
	}

	// no rules were found for the resource, default to denying access
	log.Tracef("%v:%v denied access to %v:%v:%v:%v by lack of rule (%v rules found for namespace)", logNamespace, logID, res.Namespace, res.Type, res.Name, res.Endpoint, len(s.listRules(res.Namespace)))
	return auth.ErrForbidden
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

// listRules gets all the rules from the store which match the filters.
// filters are namespace, type, name and then endpoint.
func (s *svc) listRules(filters ...string) []*pb.Rule {
	s.Lock()
	defer s.Unlock()

	var rules []*pb.Rule
	for _, r := range s.rules {
		if len(filters) > 0 && r.Resource.Namespace != filters[0] {
			continue
		}
		if len(filters) > 1 && r.Resource.Type != filters[1] {
			continue
		}
		if len(filters) > 2 && r.Resource.Name != filters[2] {
			continue
		}
		if len(filters) > 3 && r.Resource.Endpoint != filters[3] {
			continue
		}

		rules = append(rules, r)
	}

	// sort rules by priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

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
		ID:        a.Id,
		Roles:     a.Roles,
		Secret:    a.Secret,
		Metadata:  a.Metadata,
		Provider:  a.Provider,
		Namespace: a.Namespace,
	}
}
