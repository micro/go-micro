package token

import (
	"encoding/base64"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/asim/go-micro/v3/auth"
)

// authClaims to be encoded in the JWT
type authClaims struct {
	Type     string            `json:"type"`
	Scopes   []string          `json:"scopes"`
	Metadata map[string]string `json:"metadata"`

	jwt.StandardClaims
}

// JWT implementation of token provider
type JWT struct {
	opts Options
}

// New returns an initialized basic provider
func New(opts ...Option) Provider {
	return &JWT{
		opts: NewOptions(opts...),
	}
}

// Generate a new JWT
func (j *JWT) Generate(acc *auth.Account, opts ...GenerateOption) (*Token, error) {
	// decode the private key
	priv, err := base64.StdEncoding.DecodeString(j.opts.PrivateKey)
	if err != nil {
		return nil, err
	}

	// parse the private key
	key, err := jwt.ParseRSAPrivateKeyFromPEM(priv)
	if err != nil {
		return nil, ErrEncodingToken
	}

	// parse the options
	options := NewGenerateOptions(opts...)

	// generate the JWT
	expiry := time.Now().Add(options.Expiry)
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, authClaims{
		acc.Type, acc.Scopes, acc.Metadata, jwt.StandardClaims{
			Subject:   acc.ID,
			Issuer:    acc.Issuer,
			ExpiresAt: expiry.Unix(),
		},
	})
	tok, err := t.SignedString(key)
	if err != nil {
		return nil, err
	}

	// return the token
	return &Token{
		Token:   tok,
		Expiry:  expiry,
		Created: time.Now(),
	}, nil
}

// Inspect a JWT
func (j *JWT) Inspect(t string) (*auth.Account, error) {
	// decode the public key
	pub, err := base64.StdEncoding.DecodeString(j.opts.PublicKey)
	if err != nil {
		return nil, err
	}

	// parse the public key
	res, err := jwt.ParseWithClaims(t, &authClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwt.ParseRSAPublicKeyFromPEM(pub)
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	// validate the token
	if !res.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := res.Claims.(*authClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// return the token
	return &auth.Account{
		ID:       claims.Subject,
		Issuer:   claims.Issuer,
		Type:     claims.Type,
		Scopes:   claims.Scopes,
		Metadata: claims.Metadata,
	}, nil
}

// String returns JWT
func (j *JWT) String() string {
	return "jwt"
}
