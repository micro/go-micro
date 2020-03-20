package jwt

import (
	"encoding/base64"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/micro/go-micro/v2/auth/token"
)

// authClaims to be encoded in the JWT
type authClaims struct {
	Roles    []string          `json:"roles"`
	Metadata map[string]string `json:"metadata"`

	jwt.StandardClaims
}

// JWT implementation of token provider
type JWT struct {
	opts token.Options
}

// NewTokenProvider returns an initialized basic provider
func NewTokenProvider(opts ...token.Option) token.Provider {
	return &JWT{
		opts: token.NewOptions(opts...),
	}
}

// Generate a new JWT
func (j *JWT) Generate(subject string, opts ...token.GenerateOption) (*token.Token, error) {
	// decode the private key
	priv, err := base64.StdEncoding.DecodeString(j.opts.PrivateKey)
	if err != nil {
		return nil, err
	}

	// parse the private key
	key, err := jwt.ParseRSAPrivateKeyFromPEM(priv)
	if err != nil {
		return nil, token.ErrEncodingToken
	}

	// parse the options
	options := token.NewGenerateOptions(opts...)

	// generate the JWT
	expiry := time.Now().Add(options.Expiry)
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, authClaims{
		options.Roles, options.Metadata, jwt.StandardClaims{
			Subject:   subject,
			ExpiresAt: expiry.Unix(),
		},
	})
	tok, err := t.SignedString(key)
	if err != nil {
		return nil, err
	}

	// return the token
	return &token.Token{
		Subject:  subject,
		Token:    tok,
		Type:     j.String(),
		Created:  time.Now(),
		Expiry:   expiry,
		Roles:    options.Roles,
		Metadata: options.Metadata,
	}, nil
}

// Inspect a JWT
func (j *JWT) Inspect(t string) (*token.Token, error) {
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
		return nil, token.ErrInvalidToken
	}

	// validate the token
	if !res.Valid {
		return nil, token.ErrInvalidToken
	}
	claims, ok := res.Claims.(*authClaims)
	if !ok {
		return nil, token.ErrInvalidToken
	}

	// return the token
	return &token.Token{
		Token:    t,
		Subject:  claims.Subject,
		Metadata: claims.Metadata,
		Roles:    claims.Roles,
	}, nil
}

// String returns JWT
func (j *JWT) String() string {
	return "jwt"
}
