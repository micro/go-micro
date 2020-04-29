package jwt

import (
	"sync"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/auth/token"
	jwtToken "github.com/micro/go-micro/v2/auth/token/jwt"
)

// NewAuth returns a new instance of the Auth service
func NewAuth(opts ...auth.Option) auth.Auth {
	j := new(jwt)
	j.Init(opts...)
	return j
}

type rule struct {
	role     string
	resource *auth.Resource
}

type jwt struct {
	options auth.Options
	jwt     token.Provider
	rules   []*rule

	sync.Mutex
}

func (j *jwt) String() string {
	return "jwt"
}

func (j *jwt) Init(opts ...auth.Option) {
	j.Lock()
	defer j.Unlock()

	for _, o := range opts {
		o(&j.options)
	}

	if len(j.options.Namespace) == 0 {
		j.options.Namespace = auth.DefaultNamespace
	}

	j.jwt = jwtToken.NewTokenProvider(
		token.WithPrivateKey(j.options.PrivateKey),
		token.WithPublicKey(j.options.PublicKey),
	)
}

func (j *jwt) Options() auth.Options {
	j.Lock()
	defer j.Unlock()
	return j.options
}

func (j *jwt) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	options := auth.NewGenerateOptions(opts...)
	account := &auth.Account{
		ID:        id,
		Type:      options.Type,
		Roles:     options.Roles,
		Provider:  options.Provider,
		Metadata:  options.Metadata,
		Namespace: options.Namespace,
	}

	// generate a JWT secret which can be provided to the Token() method
	// and exchanged for an access token
	secret, err := j.jwt.Generate(account)
	if err != nil {
		return nil, err
	}
	account.Secret = secret.Token

	// return the account
	return account, nil
}

func (j *jwt) Grant(role string, res *auth.Resource) error {
	j.Lock()
	defer j.Unlock()
	j.rules = append(j.rules, &rule{role, res})
	return nil
}

func (j *jwt) Revoke(role string, res *auth.Resource) error {
	j.Lock()
	defer j.Unlock()

	rules := make([]*rule, 0, len(j.rules))

	var ruleFound bool
	for _, r := range rules {
		if r.role == role && r.resource == res {
			ruleFound = true
		} else {
			rules = append(rules, r)
		}
	}

	if !ruleFound {
		return auth.ErrNotFound
	}

	j.rules = rules
	return nil
}

func (j *jwt) Verify(acc *auth.Account, res *auth.Resource) error {
	j.Lock()
	if len(res.Namespace) == 0 {
		res.Namespace = j.options.Namespace
	}
	rules := j.rules
	j.Unlock()

	for _, rule := range rules {
		// validate the rule applies to the requested resource
		if rule.resource.Namespace != "*" && rule.resource.Namespace != res.Namespace {
			continue
		}
		if rule.resource.Type != "*" && rule.resource.Type != res.Type {
			continue
		}
		if rule.resource.Name != "*" && rule.resource.Name != res.Name {
			continue
		}
		if rule.resource.Endpoint != "*" && rule.resource.Endpoint != res.Endpoint {
			continue
		}

		// a blank role indicates anyone can access the resource, even without an account
		if rule.role == "" {
			return nil
		}

		// all furter checks require an account
		if acc == nil {
			continue
		}

		// this rule allows any account access, allow the request
		if rule.role == "*" {
			return nil
		}

		// if the account has the necessary role, allow the request
		for _, r := range acc.Roles {
			if r == rule.role {
				return nil
			}
		}
	}

	// no rules matched, forbid the request
	return auth.ErrForbidden
}

func (j *jwt) Inspect(token string) (*auth.Account, error) {
	return j.jwt.Inspect(token)
}

func (j *jwt) Token(opts ...auth.TokenOption) (*auth.Token, error) {
	options := auth.NewTokenOptions(opts...)

	secret := options.RefreshToken
	if len(options.Secret) > 0 {
		secret = options.Secret
	}

	account, err := j.jwt.Inspect(secret)
	if err != nil {
		return nil, err
	}

	tok, err := j.jwt.Generate(account, token.WithExpiry(options.Expiry))
	if err != nil {
		return nil, err
	}

	return &auth.Token{
		Created:      tok.Created,
		Expiry:       tok.Expiry,
		AccessToken:  tok.Token,
		RefreshToken: tok.Token,
	}, nil
}
