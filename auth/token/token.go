package token

import (
	"errors"

	"github.com/micro/go-micro/v2/auth"
)

var (
	// ErrNotFound is returned when a token cannot be found
	ErrNotFound = errors.New("token not found")
	// ErrEncodingToken is returned when the service encounters an error during encoding
	ErrEncodingToken = errors.New("error encoding the token")
	// ErrInvalidToken is returned when the token provided is not valid
	ErrInvalidToken = errors.New("invalid token provided")
)

// Provider generates and inspects tokens
type Provider interface {
	Generate(subject string, opts ...GenerateOption) (*auth.Token, error)
	Inspect(token string) (*auth.Token, error)
	String() string
}
