package jwt

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/micro/go-micro/auth"
)

// ErrInvalidPrivateKey is returned when the service provided an invalid private key
var ErrInvalidPrivateKey = errors.New("An invalid private key was provided")

// ErrEncodingToken is returned when the service encounters an error during encoding
var ErrEncodingToken = errors.New("An error occured while encoding the JWT")

// ErrInvalidToken is returned when the token provided is not valid
var ErrInvalidToken = errors.New("An invalid token was provided")

// NewAuth returns a new instance of the Auth service
func NewAuth(opts ...auth.Option) auth.Auth {
	svc := new(svc)
	svc.Init(opts...)
	return svc
}

// svc is the JWT implementation of the Auth interface
type svc struct {
	options auth.Options
}

func (s *svc) Init(opts ...auth.Option) error {
	for _, o := range opts {
		o(&s.options)
	}

	return nil
}

// AuthClaims to be encoded in the JWT
type AuthClaims struct {
	Roles    []*auth.Role      `json:"roles"`
	Metadata map[string]string `json:"metadata"`

	jwt.StandardClaims
}

// Generate a new JWT
func (s *svc) Generate(sa *auth.Account) (*auth.Account, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(s.options.PrivateKey)
	if err != nil {
		return nil, ErrEncodingToken
	}

	account := jwt.NewWithClaims(jwt.SigningMethodRS256, AuthClaims{
		sa.Roles, sa.Metadata, jwt.StandardClaims{
			Subject:   "TODO",
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
		},
	})

	sa.Token, err = account.SignedString(key)
	if err != nil {
		return nil, err
	}

	return sa, nil
}

// Revoke an authorization account
func (s *svc) Revoke(token string) error {
	return nil
}

// Validate a JWT
func (s *svc) Validate(token string) (*auth.Account, error) {
	res, err := jwt.ParseWithClaims(token, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwt.ParseRSAPublicKeyFromPEM(s.options.PublicKey)
	})
	if err != nil {
		return nil, err
	}

	if !res.Valid {
		return nil, ErrInvalidToken
	}

	claims := res.Claims.(*AuthClaims)

	return &auth.Account{
		Metadata: claims.Metadata,
		Roles:    claims.Roles,
	}, nil
}
