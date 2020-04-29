package jwt

import (
	"errors"

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

type jwt struct {
	options auth.Options
	jwt     token.Provider
}

func (j *jwt) String() string {
	return "jwt"
}

func (j *jwt) Init(opts ...auth.Option) {
	for _, o := range opts {
		o(&j.options)
	}

	j.jwt = jwtToken.NewTokenProvider(
		token.WithPrivateKey(j.options.PrivateKey),
		token.WithPublicKey(j.options.PublicKey),
	)
}

func (j *jwt) Options() auth.Options {
	return j.options
}

func (j *jwt) Generate(id string, opts ...auth.GenerateOption) (*auth.Account, error) {
	return nil, errors.New("JWT does not support Generate, use the Token method")
}

func (j *jwt) Grant(role string, res *auth.Resource) error {
	return errors.New("JWT does not support Grant")
}

func (j *jwt) Revoke(role string, res *auth.Resource) error {
	return errors.New("JWT does not support Revoke")
}

func (j *jwt) Verify(acc *auth.Account, res *auth.Resource) error {
	if acc == nil {
		return auth.ErrForbidden
	}
	return nil
}

func (j *jwt) Inspect(token string) (*auth.Account, error) {
	return j.jwt.Inspect(token)
}

func (j *jwt) Token(opts ...auth.TokenOption) (*auth.Token, error) {
	options := auth.NewTokenOptions(opts...)
	account := &auth.Account{
		ID: options.ID,
	}

	tok, err := j.jwt.Generate(account, token.WithExpiry(options.Expiry))
	if err != nil {
		return nil, err
	}

	return &auth.Token{
		Created:     tok.Created,
		Expiry:      tok.Expiry,
		AccessToken: tok.Token,
	}, nil
}
