package jwt

import (
	"sync"
	"time"

	"github.com/asim/go-micro/v3/auth"
	"github.com/asim/go-micro/v3/cmd"
	jwtToken "github.com/asim/go-micro/plugins/auth/jwt/v3/token"
)

func init() {
	cmd.DefaultAuths["jwt"] = NewAuth
}

// NewAuth returns a new instance of the Auth service
func NewAuth(opts ...auth.Option) auth.Auth {
	j := new(jwt)
	j.Init(opts...)
	return j
}

func NewRules() auth.Rules {
	return new(jwtRules)
}

type jwt struct {
	sync.Mutex
	options auth.Options
	jwt     jwtToken.Provider
}

type jwtRules struct {
	sync.Mutex
	rules []*auth.Rule
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

	j.jwt = jwtToken.New(
		jwtToken.WithPrivateKey(j.options.PrivateKey),
		jwtToken.WithPublicKey(j.options.PublicKey),
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
		ID:       id,
		Type:     options.Type,
		Scopes:   options.Scopes,
		Metadata: options.Metadata,
		Issuer:   j.Options().Namespace,
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

func (j *jwtRules) Grant(rule *auth.Rule) error {
	j.Lock()
	defer j.Unlock()
	j.rules = append(j.rules, rule)
	return nil
}

func (j *jwtRules) Revoke(rule *auth.Rule) error {
	j.Lock()
	defer j.Unlock()

	rules := []*auth.Rule{}
	for _, r := range j.rules {
		if r.ID != rule.ID {
			rules = append(rules, r)
		}
	}

	j.rules = rules
	return nil
}

func (j *jwtRules) Verify(acc *auth.Account, res *auth.Resource, opts ...auth.VerifyOption) error {
	j.Lock()
	defer j.Unlock()

	var options auth.VerifyOptions
	for _, o := range opts {
		o(&options)
	}

	return auth.Verify(j.rules, acc, res)
}

func (j *jwtRules) List(opts ...auth.ListOption) ([]*auth.Rule, error) {
	j.Lock()
	defer j.Unlock()
	return j.rules, nil
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

	access, err := j.jwt.Generate(account, jwtToken.WithExpiry(options.Expiry))
	if err != nil {
		return nil, err
	}

	refresh, err := j.jwt.Generate(account, jwtToken.WithExpiry(options.Expiry+time.Hour))
	if err != nil {
		return nil, err
	}

	return &auth.Token{
		Created:      access.Created,
		Expiry:       access.Expiry,
		AccessToken:  access.Token,
		RefreshToken: refresh.Token,
	}, nil
}
