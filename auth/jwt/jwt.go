package jwt

import (
	"encoding/base64"
	"errors"

	"github.com/dgrijalva/jwt-go"
	"github.com/micro/go-micro/v2/auth"
)

var (
	// ErrInvalidPrivateKey is returned when the service provided an invalid private key
	ErrInvalidPrivateKey = errors.New("An invalid private key was provided")

	// ErrEncodingToken is returned when the service encounters an error during encoding
	ErrEncodingToken = errors.New("An error occured while encoding the JWT")

	// ErrInvalidToken is returned when the token provided is not valid
	ErrInvalidToken = errors.New("An invalid token was provided")

	// ErrMissingToken is returned when no token is provided
	ErrMissingToken = errors.New("A valid JWT is required")
)

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

func (s *svc) String() string {
	return "jwt"
}

func (s *svc) Options() auth.Options {
	return s.options
}

func (s *svc) Init(opts ...auth.Option) error {
	for _, o := range opts {
		o(&s.options)
	}

	return nil
}

// AuthClaims to be encoded in the JWT
type AuthClaims struct {
	Id       string            `json:"id"`
	Roles    []*auth.Role      `json:"roles"`
	Metadata map[string]string `json:"metadata"`

	jwt.StandardClaims
}

// Generate a new JWT
func (s *svc) Generate(id string, ops ...auth.GenerateOption) (*auth.Account, error) {
	// decode the private key
	priv, err := base64.StdEncoding.DecodeString(s.options.PrivateKey)
	if err != nil {
		return nil, err
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(priv)
	if err != nil {
		return nil, ErrEncodingToken
	}

	options := auth.NewGenerateOptions(ops...)
	account := jwt.NewWithClaims(jwt.SigningMethodRS256, AuthClaims{
		id, options.Roles, options.Metadata, jwt.StandardClaims{
			Subject:   id,
			ExpiresAt: options.Expiry.Unix(),
		},
	})

	token, err := account.SignedString(key)
	if err != nil {
		return nil, err
	}

	return &auth.Account{
		Id:       id,
		Token:    token,
		Roles:    options.Roles,
		Metadata: options.Metadata,
	}, nil
}

// Revoke an authorization account
func (s *svc) Revoke(token string) error {
	return nil
}

// Verify a JWT
func (s *svc) Verify(token string) (*auth.Account, error) {
	if token == "" {
		return nil, ErrMissingToken
	}

	// decode the public key
	pub, err := base64.StdEncoding.DecodeString(s.options.PublicKey)
	if err != nil {
		return nil, err
	}

	res, err := jwt.ParseWithClaims(token, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwt.ParseRSAPublicKeyFromPEM(pub)
	})
	if err != nil {
		return nil, err
	}

	if !res.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := res.Claims.(*AuthClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return &auth.Account{
		Id:       claims.Id,
		Metadata: claims.Metadata,
		Roles:    claims.Roles,
	}, nil
}
